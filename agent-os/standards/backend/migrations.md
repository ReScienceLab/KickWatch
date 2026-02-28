## Database migration best practices

- **Reversible Migrations**: Always implement rollback/down methods to enable safe migration reversals when using migration tools
- **Small, Focused Changes**: Keep each migration focused on a single logical change for clarity and easier troubleshooting
- **Zero-Downtime Deployments**: Add new columns as nullable first, backfill data, then set NOT NULL for production safety
- **Separate Schema and Data**: Keep schema changes separate from data migrations for better rollback safety
- **Index Management**: Create indexes on large tables carefully, using CONCURRENTLY when available to avoid locks
- **Naming Conventions**: Use clear, descriptive names that indicate what the migration does
- **Version Control**: Always commit migrations to version control and never modify existing migrations after deployment
- **GORM AutoMigrate Order**: Run ALL manual SQL migrations (column renames, data fixes) BEFORE AutoMigrate in db.Init()
- **CRITICAL**: GORM AutoMigrate never renames columns - it creates NEW columns if field names don't match existing columns
