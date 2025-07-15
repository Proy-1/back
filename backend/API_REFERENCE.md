# API Documentation - E-Commerce Backend

## Base URL
```
http://localhost:5000
```

## All Available Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/health | Health check |
| GET | /api/products | Get all products |
| POST | /api/products | Create product |
| GET | /api/products/{id} | Get single product |
| PUT | /api/products/{id} | Update product |
| DELETE | /api/products/{id} | Delete product |
| GET | /api/admins | Get all admins |
| POST | /api/admins | Create admin |
| DELETE | /api/admins/{id} | Delete admin |
| GET | /api/login | Login info |
| POST | /api/login | Login admin |
| POST | /api/register | Register admin |
| POST | /api/upload | Upload image |
| GET | /api/stats | Get statistics |
| GET | /static/uploads/{filename} | Serve uploaded files |

## Frontend Integration

All endpoints are ready for dashboard integration. CORS is configured for:
- http://localhost:3000
- http://localhost:8080

## Example Usage

### JavaScript Fetch
```javascript
// Get all products
fetch('http://localhost:5000/api/products')
  .then(response => response.json())
  .then(data => console.log(data));

// Create product
fetch('http://localhost:5000/api/products', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    name: 'New Product',
    price: 50000,
    description: 'Description',
    image_url: '/static/uploads/image.jpg'
  })
});

// Update product
fetch('http://localhost:5000/api/products/PRODUCT_ID', {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    name: 'Updated Name',
    price: 75000
  })
});

// Delete product
fetch('http://localhost:5000/api/products/PRODUCT_ID', {
  method: 'DELETE'
});
```

## Response Format

All endpoints return JSON with either:
- **Success**: Data object or array
- **Error**: `{"error": "Error message"}`
