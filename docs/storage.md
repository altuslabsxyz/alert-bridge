# Storage Options

Alert Bridge supports three storage backends, each optimized for different use cases.

## In-Memory Storage

Fast but ephemeral - data is lost on restart.

### Configuration

```yaml
storage:
  type: memory
```

### Use Cases

- Development and testing
- Stateless deployments
- When persistence is not required

### Limitations

- No data persistence
- Single instance only
- Data lost on restart

## SQLite Storage

Persistent storage with excellent performance, recommended for single-instance deployments.

### Configuration

```yaml
storage:
  type: sqlite
  sqlite:
    path: ./data/alert-bridge.db
```

### Features

- Data persists across restarts
- Sub-millisecond read operations (15.8µs average)
- Concurrent read support via WAL mode
- Automatic schema migrations
- Foreign key constraints and data integrity
- Graceful shutdown with WAL checkpoint

### Performance

- Read operations: ~15.8µs (0.0158ms)
- Write operations: Sub-50ms
- Concurrent operations: 100+ simultaneous reads

### Production Considerations

- Ensure the data directory exists and is writable
- Regular backups recommended for production
- Database file will grow with alert volume
- Consider log rotation for WAL files
- **Single instance only** - SQLite uses file-based locking

### Database Management

#### View Schema

```bash
sqlite3 ./data/alert-bridge.db ".schema"
```

#### Query Alerts

```bash
sqlite3 ./data/alert-bridge.db "SELECT id, name, state FROM alerts;"
```

#### Backup Database

```bash
# Create a backup
sqlite3 ./data/alert-bridge.db ".backup ./data/alert-bridge-backup.db"

# Or use standard file copy (when app is stopped)
cp ./data/alert-bridge.db ./data/alert-bridge-backup.db
```

#### Compact Database

```bash
sqlite3 ./data/alert-bridge.db "VACUUM;"
```

## MySQL Storage

Scalable persistent storage with multi-instance support for high availability and production deployments.

### Configuration

```yaml
storage:
  type: mysql
  mysql:
    primary:
      host: mysql.example.com
      port: 3306
      database: alert_bridge
      username: alert_bridge_user
      password: ${MYSQL_PASSWORD}

    # Optional: for read scaling
    replica:
      enabled: false
      host: mysql-replica.example.com
      port: 3306
      database: alert_bridge
      username: alert_bridge_reader
      password: ${MYSQL_REPLICA_PASSWORD}

    pool:
      max_open_conns: 25      # Maximum open connections
      max_idle_conns: 5       # Maximum idle connections
      conn_max_lifetime: 3m   # Maximum connection lifetime
      conn_max_idle_time: 1m  # Maximum idle time

    timeout: 5s               # Query timeout
    parse_time: true          # Parse time values to time.Time
    charset: utf8mb4          # Character set
```

### Features

- Multi-instance deployment support (3+ concurrent instances)
- Optimistic locking prevents concurrent update conflicts
- Primary-replica support for read scaling
- Connection pool with configurable limits
- Automatic schema migrations
- Foreign key constraints and referential integrity
- JSON columns for flexible label/annotation storage

### Performance

- Read operations: < 100ms target (10K alerts)
- Write operations: < 200ms target
- Cross-instance visibility: < 1 second
- Concurrent instances: 3+ supported

### Production Considerations

- Use MySQL 8.0+ or MariaDB 10.5+ with InnoDB engine
- Configure connection pool based on expected load
- Set up read replicas for scaling read operations
- Regular backups using mysqldump or physical backups
- Monitor connection pool metrics (wait count, in-use connections)
- Use separate credentials for primary (read-write) and replica (read-only)
- Enable slow query logging for queries > 100ms
- Consider partitioning for very large alert volumes

### Database Setup

#### Create Database and User

```bash
# Connect to MySQL
mysql -u root -p

# Create database
CREATE DATABASE alert_bridge CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

# Create user with appropriate privileges
CREATE USER 'alert_bridge_user'@'%' IDENTIFIED BY 'secure_password';
GRANT ALL PRIVILEGES ON alert_bridge.* TO 'alert_bridge_user'@'%';

# Optional: Create read-only user for replicas
CREATE USER 'alert_bridge_reader'@'%' IDENTIFIED BY 'reader_password';
GRANT SELECT ON alert_bridge.* TO 'alert_bridge_reader'@'%';

FLUSH PRIVILEGES;
```

### Database Management

#### View Schema

```bash
mysql -u alert_bridge_user -p alert_bridge -e "SHOW TABLES;"
mysql -u alert_bridge_user -p alert_bridge -e "DESCRIBE alerts;"
mysql -u alert_bridge_user -p alert_bridge -e "DESCRIBE ack_events;"
mysql -u alert_bridge_user -p alert_bridge -e "DESCRIBE silences;"
```

#### Query Alerts

```bash
# View all active alerts
mysql -u alert_bridge_user -p alert_bridge -e \
  "SELECT id, name, state, severity FROM alerts WHERE state = 'firing';"

# View recent acknowledgments
mysql -u alert_bridge_user -p alert_bridge -e \
  "SELECT alert_id, source, acknowledged_by_name, acknowledged_at
   FROM ack_events ORDER BY acknowledged_at DESC LIMIT 10;"

# View active silences
mysql -u alert_bridge_user -p alert_bridge -e \
  "SELECT id, instance, fingerprint, start_at, end_at
   FROM silences WHERE start_at <= NOW() AND end_at > NOW();"
```

#### Backup Database

```bash
# Create a backup using mysqldump
mysqldump -u alert_bridge_user -p alert_bridge > alert-bridge-backup.sql

# Create compressed backup
mysqldump -u alert_bridge_user -p alert_bridge | gzip > alert-bridge-backup-$(date +%Y%m%d).sql.gz

# Backup specific tables
mysqldump -u alert_bridge_user -p alert_bridge alerts ack_events silences > backup.sql
```

#### Restore Database

```bash
# Restore from backup
mysql -u alert_bridge_user -p alert_bridge < alert-bridge-backup.sql

# Restore from compressed backup
gunzip < alert-bridge-backup-20250101.sql.gz | mysql -u alert_bridge_user -p alert_bridge
```

#### Monitor Connection Pool

```bash
# View current connections
mysql -u root -p -e "SHOW PROCESSLIST;"

# View connection statistics
mysql -u root -p -e "SHOW STATUS LIKE 'Threads%';"
mysql -u root -p -e "SHOW STATUS LIKE 'Connections';"
mysql -u root -p -e "SHOW STATUS LIKE 'Max_used_connections';"

# View table statistics
mysql -u alert_bridge_user -p alert_bridge -e \
  "SELECT COUNT(*) as total_alerts FROM alerts;"
mysql -u alert_bridge_user -p alert_bridge -e \
  "SELECT COUNT(*) as total_acks FROM ack_events;"
```

#### Clean Up Old Data

```bash
# Delete old resolved alerts (older than 30 days)
mysql -u alert_bridge_user -p alert_bridge -e \
  "DELETE FROM alerts WHERE state = 'resolved'
   AND updated_at < DATE_SUB(NOW(), INTERVAL 30 DAY);"

# Delete expired silences
mysql -u alert_bridge_user -p alert_bridge -e \
  "DELETE FROM silences WHERE end_at < NOW();"

# Optimize tables after cleanup
mysql -u alert_bridge_user -p alert_bridge -e "OPTIMIZE TABLE alerts;"
mysql -u alert_bridge_user -p alert_bridge -e "OPTIMIZE TABLE ack_events;"
mysql -u alert_bridge_user -p alert_bridge -e "OPTIMIZE TABLE silences;"
```

## Migration from SQLite to MySQL

1. Export data from SQLite using `.dump` command
2. Create MySQL database and user
3. Update configuration to use MySQL storage
4. Restart application (migrations run automatically)
5. Verify data integrity and performance

## Comparison

| Feature | Memory | SQLite | MySQL |
|---------|--------|--------|-------|
| Persistence | No | Yes | Yes |
| Multi-instance | No | No | Yes |
| Performance | Fastest | Very Fast | Fast |
| Setup Complexity | None | Low | Medium |
| Recommended For | Dev/Test | Single instance | Multi-instance/HA |
| Data Recovery | None | File backup | Full backup tools |
| Scalability | Limited | Limited | High |

## Next Steps

- [Deployment Guide](deployment.md) - Deploy with your chosen storage
- [Troubleshooting](troubleshooting.md) - Common storage issues
