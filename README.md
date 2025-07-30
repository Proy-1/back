# Go Backend - E-Commerce Dashboard

Backend API server yang dibuat dengan Go dan Gin framework untuk dashboard e-commerce.

## 🚀 Quick Start

### Prerequisites
- Go 1.21 atau lebih baru
- MongoDB Atlas (Cloud) atau MongoDB Local

### Installation

1. **Clone repository:**
   ```bash
   git clone https://github.com/Proy-1/back.git
   cd back/backend
   ```

2. **Install dependencies:**
   ```bash
   go mod tidy
   ```

3. **Setup Database:**
   
   **Option A: MongoDB Atlas (Recommended - Works Anywhere)**
   ```bash
   # Run atlas setup guide
   atlas-guide.bat
   
   # Then update connection string
   setup-atlas.bat
   ```
   
   **Option B: Local MongoDB**
   ```bash
   # Windows
   net start mongodb
   
   # macOS/Linux
   sudo systemctl start mongod
   ```

4. **Run backend:**
   ```bash
   # Development
   go run main.go
   
   # Production
   go build -o backend.exe .
   ./backend.exe
   ```

### Windows Quick Setup
```bash
# Setup dependencies dan run tests
setup.bat

# Start server
start.bat
```

## 📋 API Endpoints

### Health Check
- `GET /api/health` - Cek status server dan database

### Products
- `GET /api/products` - Get all products
- `POST /api/products` - Create new product
- `GET /api/products/:id` - Get product by ID
- `PUT /api/products/:id` - Update product
- `DELETE /api/products/:id` - Delete product

### Admin Management
- `GET /api/admins` - Get all admins
- `POST /api/admins` - Create new admin
- `DELETE /api/admins/:id` - Delete admin

### Authentication
- `POST /api/login` - Login admin
- `POST /api/register` - Register new admin

### File Upload
- `POST /api/upload` - Upload image file

### Statistics
- `GET /api/stats` - Get system statistics

## 🔧 Configuration

Environment variables (`.env`):
```env
PORT=5000
MONGO_URI=mongodb://localhost:27017/pitipaw
DEBUG=true
ENVIRONMENT=development
MAX_FILE_SIZE=10485760
UPLOAD_DIR=static/uploads
```

## 🧪 Testing

```bash
go test -v
```

##  Frontend Integration

Backend ini dirancang untuk bekerja dengan dashboard frontend:
https://github.com/Proy-1/dashboard-1

### Setup Dashboard
1. Pastikan backend running di `http://localhost:5000`
2. Clone dan buka dashboard frontend
3. Semua fitur CRUD, authentication, dan file upload akan berfungsi

##  Troubleshooting

### MongoDB Connection Issues
- Pastikan MongoDB service running
- Check connection string di `.env`

### CORS Issues
- Verify frontend URL matches allowed origins di `main.go`

### Port Already in Use
```bash
# Windows
netstat -ano | findstr :5000
taskkill /PID <PID> /F
```

## 📞 Support

Jika mengalami masalah:
1. Check server logs untuk error details
2. Verify MongoDB connection
3. Test API endpoints dengan curl atau Postman
