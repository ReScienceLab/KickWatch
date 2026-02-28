## API endpoint standards and conventions

- **RESTful Design**: Follow REST principles with clear resource-based URLs and appropriate HTTP methods (GET, POST, PUT, PATCH, DELETE)
- **Consistent Naming**: Use plural nouns for collections (`/campaigns`, `/alerts`, `/devices`) with Gin path parameters (`:pid`)
- **Versioning**: No API versioning - keep API stable and prefer additive changes only
- **Plural Nouns**: Use plural nouns for resource endpoints (e.g., `/campaigns`, `/alerts`) for consistency
- **Nested Resources**: Limit nesting depth to 2 levels maximum (e.g., `/campaigns/:pid/history`) to keep URLs readable
- **Query Parameters**: Use query parameters for filtering (`?category=games`), sorting (`?sort=newest`), and pagination (`?cursor=...&limit=20`)
- **HTTP Status Codes**: Return appropriate HTTP status codes (200 OK, 201 Created, 400 Bad Request, 404 Not Found, 500 Internal Server Error)
- **Rate Limiting Headers**: Not currently implemented but consider for future API protection
- **Error Format**: Always return JSON with `{"error": "message"}` format for consistency
- **Pagination**: Use cursor-based pagination with `next_cursor` field (nil when no more results)
