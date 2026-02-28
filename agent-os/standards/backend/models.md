## Database model best practices

- **Clear Naming**: Use singular names for models (Campaign, Alert) and plural for tables (campaigns, alerts) following GORM conventions
- **Timestamps**: Include created and updated timestamps on all tables using `gorm.Model` for auditing and debugging
- **Data Integrity**: Use database constraints (NOT NULL, UNIQUE, foreign keys) and GORM tags to enforce data rules at the database level
- **Appropriate Data Types**: Choose data types that match the data's purpose - use `gorm:"type:text"` for long strings, `gorm:"type:date"` for dates
- **Indexes on Foreign Keys**: Index foreign key columns (`gorm:"index"`) and other frequently queried fields for performance
- **Validation at Multiple Layers**: Implement validation at both model and database levels for defense in depth
- **Relationship Clarity**: Define relationships clearly with appropriate cascade behaviors and naming conventions
- **Avoid Over-Normalization**: Balance normalization with practical query performance needs
- **GORM Column Tags**: Use `gorm:"column:pid"` to override snake_case conversion for abbreviations (PID → pid, not p_id)
