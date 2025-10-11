# Where Is It? - Inventory Management System

A Go-based web application for managing items and their storage locations with hierarchical organization and powerful search capabilities.

## Features

- **Hierarchical Locations**: Organize storage locations in a tree structure (e.g., Cellar > Werkstatt > Cupboard-2)
- **Detailed Item Management**: Store items with quantities, units, properties, and tags
- **Flexible Properties**: Add custom properties to items (e.g., size: M4, length: 20mm, standard: ISO XYZ)
- **Tagging System**: Categorize items with tags for easy filtering
- **Powerful Search**: Search by name, description, properties, tags, or location
- **Web Interface**: Clean, responsive web UI for managing your inventory
- **REST API**: JSON API for programmatic access
- **Docker Ready**: Easy deployment with Docker containers
- **JSON Storage**: Simple file-based storage (easily extensible to databases)

## Quick Start

### Using Go

1. Clone and run:
```bash
go mod tidy
go run main.go
```

2. Open http://localhost:8080

### Using Docker

1. Build and run:
```bash
docker build -t where-is-it .
docker run -p 8080:8080 -v $(pwd)/data:/root/data where-is-it
```

2. Open http://localhost:8080

## Usage Example

1. **Create Locations**:
   - Cellar
   - Cellar > Werkstatt  
   - Cellar > Werkstatt > Cupboard-2

2. **Add Items**:
   - Name: "M4 ISO XYZ Screws"
   - Location: "Cellar/Werkstatt/Cupboard-2"
   - Quantity: 50 pieces
   - Properties: size=M4, length=20mm, standard=ISO XYZ
   - Tags: screws, fasteners, hardware

3. **Search**:
   - "M4" - finds all M4 items
   - "screws" - finds all screw-related items
   - "werkstatt" - finds all items in the workshop

## API Endpoints

- `GET /api/locations` - List all locations
- `POST /api/locations` - Create location
- `GET /api/items` - List all items  
- `POST /api/items` - Create item
- `POST /api/search` - Search items

## Storage

Data is stored in JSON files in the `./data` directory:
- `locations.json` - All locations
- `items.json` - All items

The storage interface is designed to be easily replaceable with database backends.

## Development

The application follows clean architecture principles:

- `internal/models/` - Data structures
- `internal/storage/` - Storage interface and implementations
- `internal/handlers/` - HTTP handlers for web and API
- `web/templates/` - HTML templates
- `web/static/` - CSS and static assets

## Docker Deployment

For production deployment with persistent data:

```bash
docker run -d \
  --name where-is-it \
  -p 8080:8080 \
  -v /path/to/data:/root/data \
  where-is-it
```

## Next Steps

- Database backend (PostgreSQL, SQLite)
- User authentication
- Image uploads for items
- Barcode/QR code scanning
- Import/export functionality
- Advanced reporting