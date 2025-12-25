# Troubleshooting Guide

## SQLite Issues

### "database is locked" error

**Symptoms:**
- Error message: "database is locked"
- Write operations fail
- Application hangs on database access

**Causes:**
- Multiple instances running simultaneously
- Stale lock files
- Another process accessing the database

**Solutions:**
1. Ensure only one instance is running:
   ```bash
   ps aux | grep alert-bridge
   killall alert-bridge
   ```

2. Check for stale lock files:
   ```bash
   ls -la ./data/
   rm ./data/alert-bridge.db-shm
   rm ./data/alert-bridge.db-wal
   ```

3. Verify no other process is accessing the database:
   ```bash
   lsof ./data/alert-bridge.db
   ```

4. Consider switching to MySQL for multi-instance deployments

### "no such table" error

**Symptoms:**
- Error on startup or first query
- Migrations not applied
- Tables don't exist

**Causes:**
- Migrations failed to run
- Database file permissions
- Corrupted database file

**Solutions:**
1. Check database file permissions:
   ```bash
   ls -la ./data/alert-bridge.db
   chmod 644 ./data/alert-bridge.db
   ```

2. Verify data directory exists and is writable:
   ```bash
   mkdir -p ./data
   chmod 755 ./data
   ```

3. Check application logs for migration errors:
   ```bash
   grep -i "migration" /var/log/alert-bridge/app.log
   ```

4. Delete and recreate database (CAUTION: loses all data):
   ```bash
   rm ./data/alert-bridge.db*
   ./alert-bridge  # Migrations will run automatically
   ```

### Performance Degradation

**Symptoms:**
- Slow queries
- High CPU usage
- Large database file

**Solutions:**
1. Run VACUUM to compact database:
   ```bash
   sqlite3 ./data/alert-bridge.db "VACUUM;"
   ```

2. Check database file size:
   ```bash
   ls -lh ./data/alert-bridge.db
   ```

3. Archive old resolved alerts:
   ```bash
   sqlite3 ./data/alert-bridge.db \
     "DELETE FROM alerts WHERE state = 'resolved' AND updated_at < datetime('now', '-30 days');"
   ```

4. Monitor WAL file size:
   ```bash
   ls -lh ./data/alert-bridge.db-wal
   ```

## MySQL Issues

### "too many connections" error

**Symptoms:**
- Error: "Too many connections"
- New connections fail
- Connection pool exhausted

**Solutions:**
1. Increase max_connections in MySQL:
   ```sql
   SET GLOBAL max_connections = 200;
   ```

2. Reduce max_open_conns in application config:
   ```yaml
   storage:
     mysql:
       pool:
         max_open_conns: 15  # Reduce from 25
   ```

3. Check for connection leaks:
   ```sql
   SHOW PROCESSLIST;
   ```

4. Monitor connection pool metrics in logs

### "Lock wait timeout exceeded" error

**Symptoms:**
- Error during write operations
- Long-running transactions
- Deadlocks

**Solutions:**
1. Check for deadlocks:
   ```sql
   SHOW ENGINE INNODB STATUS\G
   ```

2. Verify READ COMMITTED isolation level:
   ```sql
   SELECT @@transaction_isolation;
   ```

3. Reduce transaction scope in application code

4. Check for long-running queries:
   ```sql
   SELECT * FROM information_schema.PROCESSLIST
   WHERE TIME > 5;
   ```

### "connection refused" or "can't connect" error

**Symptoms:**
- Cannot connect to MySQL
- Timeout on startup
- Network errors

**Solutions:**
1. Verify MySQL is running:
   ```bash
   mysql -u alert_bridge_user -p -e "SELECT 1;"
   ```

2. Check host and port in configuration:
   ```yaml
   storage:
     mysql:
       primary:
         host: mysql.example.com  # Verify correct hostname
         port: 3306                # Verify correct port
   ```

3. Test network connectivity:
   ```bash
   telnet mysql.example.com 3306
   ping mysql.example.com
   ```

4. Verify user permissions:
   ```sql
   SHOW GRANTS FOR 'alert_bridge_user'@'%';
   ```

5. Check firewall rules allow connection

### Slow Query Performance

**Symptoms:**
- Queries taking > 100ms
- High database CPU usage
- Slow response times

**Solutions:**
1. Enable slow query log:
   ```sql
   SET GLOBAL slow_query_log = 'ON';
   SET GLOBAL long_query_time = 0.1;  -- 100ms threshold
   ```

2. Analyze slow queries:
   ```bash
   mysqldumpslow /var/log/mysql/slow-query.log
   ```

3. Check if indexes exist:
   ```sql
   SHOW INDEX FROM alerts;
   SHOW INDEX FROM ack_events;
   SHOW INDEX FROM silences;
   ```

4. Monitor table sizes:
   ```sql
   SELECT
     table_name,
     round(((data_length + index_length) / 1024 / 1024), 2) AS "Size (MB)"
   FROM information_schema.TABLES
   WHERE table_schema = 'alert_bridge';
   ```

5. Consider partitioning for very large tables

### Replication Lag (Primary-Replica)

**Symptoms:**
- Stale data on reads
- Seconds_Behind_Master > 1
- Inconsistent query results

**Solutions:**
1. Check replication status:
   ```sql
   SHOW SLAVE STATUS\G
   ```

2. Monitor lag metric:
   ```sql
   SELECT Seconds_Behind_Master;
   ```

3. Increase replica server resources (CPU, memory, I/O)

4. Check network latency:
   ```bash
   ping mysql-replica.example.com
   ```

5. Verify binary log size and retention:
   ```sql
   SHOW BINARY LOGS;
   ```

### "Deadlock found when trying to get lock" error

**Symptoms:**
- Occasional write failures
- Transaction rolled back
- Concurrent update conflicts

**Solutions:**
1. This is expected behavior - application retries automatically

2. If frequent, review transaction patterns in logs

3. Consider breaking large transactions into smaller ones

4. Monitor deadlock frequency:
   ```sql
   SHOW ENGINE INNODB STATUS\G
   ```

## Application Issues

### Cannot Connect to Slack

**Symptoms:**
- Slack messages not sent
- Error: "invalid_auth" or "not_authed"
- Connection timeout

**Solutions:**
1. Verify bot token is correct:
   ```yaml
   slack:
     bot_token: xoxb-...  # Must start with xoxb-
   ```

2. Check token scopes in Slack App settings:
   - Required: `chat:write`, `chat:write.public`, `reactions:write`

3. Test token manually:
   ```bash
   curl -X POST https://slack.com/api/auth.test \
     -H "Authorization: Bearer xoxb-your-token"
   ```

4. Verify network connectivity:
   ```bash
   ping slack.com
   curl -I https://slack.com/api/
   ```

### Cannot Connect to PagerDuty

**Symptoms:**
- Incidents not created
- Error: "Unauthorized" or "Invalid API key"
- Connection timeout

**Solutions:**
1. Verify API token is correct:
   ```yaml
   pagerduty:
     api_token: your-token  # From PagerDuty API Access Keys
   ```

2. Check routing key for Events API v2:
   ```yaml
   pagerduty:
     routing_key: R123...  # From Integration settings
   ```

3. Test API manually:
   ```bash
   curl -X GET https://api.pagerduty.com/users \
     -H "Authorization: Token token=your-token" \
     -H "Accept: application/vnd.pagerduty+json;version=2"
   ```

4. Verify network connectivity:
   ```bash
   ping events.pagerduty.com
   ```

### Alerts Not Persisting

**Symptoms:**
- Data lost on restart
- Empty database after restart
- Alerts not saved

**Solutions:**
1. Verify storage type is not "memory":
   ```yaml
   storage:
     type: sqlite  # or mysql, not "memory"
   ```

2. Check database path exists:
   ```bash
   ls -la ./data/alert-bridge.db
   ```

3. Verify data directory is writable:
   ```bash
   chmod 755 ./data
   ```

4. Check logs for write errors:
   ```bash
   grep -i "error.*save\|error.*write" /var/log/alert-bridge/app.log
   ```

### High Memory Usage

**Symptoms:**
- Memory usage growing continuously
- OOM (Out of Memory) errors
- Container restarts

**Solutions:**
1. Check for memory leaks in logs

2. Monitor alert volume:
   ```bash
   # For SQLite
   sqlite3 ./data/alert-bridge.db "SELECT COUNT(*) FROM alerts;"

   # For MySQL
   mysql -u alert_bridge_user -p alert_bridge \
     -e "SELECT COUNT(*) FROM alerts;"
   ```

3. Archive old resolved alerts

4. Increase memory limits in deployment:
   ```yaml
   resources:
     limits:
       memory: "512Mi"  # Increase if needed
   ```

### Configuration Not Loading

**Symptoms:**
- Using default values instead of config
- Environment variables not substituted
- Config file not found

**Solutions:**
1. Verify CONFIG_PATH environment variable:
   ```bash
   echo $CONFIG_PATH
   export CONFIG_PATH=/path/to/config.yaml
   ```

2. Check file path is correct:
   ```bash
   ls -la /path/to/config.yaml
   ```

3. Verify YAML syntax:
   ```bash
   # Install yamllint
   pip install yamllint

   # Validate config
   yamllint config/config.yaml
   ```

4. Check environment variable substitution:
   ```bash
   # Ensure variables are set
   echo $SLACK_BOT_TOKEN
   echo $MYSQL_PASSWORD
   ```

## Getting Help

If you're still experiencing issues:

1. Check application logs for detailed error messages
2. Enable debug logging in configuration
3. Review the [Architecture](architecture.md) to understand system behavior
4. Search existing GitHub issues
5. Create a new issue with:
   - Description of the problem
   - Steps to reproduce
   - Configuration (redact secrets!)
   - Relevant log excerpts
   - Environment details (OS, Go version, database version)

## Next Steps

- [Storage](storage.md) - Configure persistent storage
- [Deployment](deployment.md) - Deploy to production
- [Development](development.md) - Contribute fixes
