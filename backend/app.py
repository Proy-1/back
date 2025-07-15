from flask import Flask, jsonify, request, send_from_directory
from flask_cors import CORS
from pymongo import MongoClient
import os
from dotenv import load_dotenv
from bson.objectid import ObjectId
from werkzeug.utils import secure_filename
from werkzeug.security import generate_password_hash, check_password_hash

# Load environment variables from .env file
load_dotenv()

app = Flask(__name__)
# Configure file upload size limit (10MB)
app.config['MAX_CONTENT_LENGTH'] = 10 * 1024 * 1024  # 10MB

# Configure CORS more explicitly for all routes including static files
# Port 3000: repo front, Port 8000: repo dashboard, Port 5000: repo back (this backend)
CORS(app, origins=["http://localhost:3000", "http://localhost:8000", "http://127.0.0.1:3000", "http://127.0.0.1:8000"], 
     resources={
         r"/api/*": {"origins": ["http://localhost:3000", "http://localhost:8000", "http://127.0.0.1:3000", "http://127.0.0.1:8000"]},
         r"/static/*": {"origins": ["http://localhost:3000", "http://localhost:8000", "http://127.0.0.1:3000", "http://127.0.0.1:8000"]}
     })

# MongoDB connection
MONGO_URI = os.getenv('MONGO_URI', 'mongodb://localhost:27017/pitipaw')
client = MongoClient(MONGO_URI)
db = client.get_default_database()

UPLOAD_FOLDER = 'static/uploads'
ALLOWED_EXTENSIONS = {'png', 'jpg', 'jpeg', 'gif'}
MAX_FILE_SIZE = 10 * 1024 * 1024  # 10MB in bytes
app.config['UPLOAD_FOLDER'] = UPLOAD_FOLDER

if not os.path.exists(UPLOAD_FOLDER):
    os.makedirs(UPLOAD_FOLDER)

def allowed_file(filename):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS

def parse_product(product):
    return {
        "_id": str(product.get("_id")),
        "name": product.get("name"),
        "price": product.get("price"),
        "description": product.get("description"),
        "image_url": product.get("image_url"),
    }

# Error handler for file too large
@app.errorhandler(413)
def request_entity_too_large(error):
    return jsonify({'error': 'File terlalu besar. Maksimal 10MB'}), 413

# HEALTH CHECK
@app.route('/api/health')
def health_check():
    try:
        # Test database connection
        db.products.count_documents({})
        return jsonify({'status': 'ok', 'message': 'Backend is running', 'database': 'connected'})
    except Exception as e:
        return jsonify({'status': 'error', 'message': 'Database connection failed', 'error': str(e)}), 500

# UPLOAD ENDPOINTS
@app.route('/api/upload', methods=['POST'])
def upload_image():
    if 'image' not in request.files:
        return jsonify({'error': 'No file part'}), 400
    file = request.files['image']
    if file.filename == '':
        return jsonify({'error': 'No selected file'}), 400
    
    # Check file size manually (additional validation)
    file.seek(0, os.SEEK_END)
    file_size = file.tell()
    file.seek(0)  # Reset file pointer
    
    if file_size > MAX_FILE_SIZE:
        return jsonify({'error': f'File terlalu besar. Maksimal 10MB (ukuran file: {file_size / (1024*1024):.1f}MB)'}), 400
    
    if file and allowed_file(file.filename):
        filename = secure_filename(file.filename)
        filepath = os.path.join(app.config['UPLOAD_FOLDER'], filename)
        file.save(filepath)
        url = f'/static/uploads/{filename}'
        return jsonify({'image_url': url, 'file_size': f'{file_size / (1024*1024):.1f}MB'}), 201
    return jsonify({'error': 'File not allowed'}), 400

# Serve static files with CORS headers
@app.route('/static/uploads/<path:filename>')
def serve_static(filename):
    response = send_from_directory(app.config['UPLOAD_FOLDER'], filename)
    # Add CORS headers manually for static files
    response.headers['Access-Control-Allow-Origin'] = '*'
    response.headers['Access-Control-Allow-Methods'] = 'GET'
    response.headers['Access-Control-Allow-Headers'] = 'Content-Type'
    return response

# PRODUCT CRUD ENDPOINTS

# CREATE: Tambah produk baru
@app.route('/api/products', methods=['POST'])
def create_product():
    try:
        data = request.json
        
        if not data:
            return jsonify({"error": "Data tidak boleh kosong"}), 400
            
        # Validate required fields
        if not data.get("name") or not data.get("price"):
            return jsonify({"error": "Nama dan harga produk wajib diisi"}), 400
            
        result = db.products.insert_one({
            "name": data.get("name"),
            "price": data.get("price"),
            "description": data.get("description", ""),
            "image_url": data.get("image_url", "")
        })
        
        new_product = db.products.find_one({"_id": result.inserted_id})
        return jsonify(parse_product(new_product)), 201
        
    except Exception as e:
        return jsonify({"error": f"Error creating product: {str(e)}"}), 500

# READ: Ambil semua produk
@app.route('/api/products', methods=['GET'])
def get_products():
    try:
        products = db.products.find()
        return jsonify([parse_product(p) for p in products])
    except Exception as e:
        return jsonify({"error": f"Error fetching products: {str(e)}"}), 500

# READ: Ambil satu produk berdasarkan id
@app.route('/api/products/<product_id>', methods=['GET'])
def get_product(product_id):
    try:
        product = db.products.find_one({"_id": ObjectId(product_id)})
        if not product:
            return jsonify({"error": "Produk tidak ditemukan"}), 404
        return jsonify(parse_product(product))
    except Exception as e:
        return jsonify({"error": f"Error fetching product: {str(e)}"}), 500

# UPDATE: Ubah data produk
@app.route('/api/products/<product_id>', methods=['PUT'])
def update_product(product_id):
    try:
        data = request.json
        if not data:
            return jsonify({"error": "Data tidak boleh kosong"}), 400
            
        update_data = {}
        if data.get("name") is not None:
            update_data["name"] = data.get("name")
        if data.get("price") is not None:
            update_data["price"] = data.get("price")
        if data.get("description") is not None:
            update_data["description"] = data.get("description")
        if data.get("image_url") is not None:
            update_data["image_url"] = data.get("image_url")
            
        if not update_data:
            return jsonify({"error": "Tidak ada data untuk diupdate"}), 400
            
        result = db.products.update_one(
            {"_id": ObjectId(product_id)},
            {"$set": update_data}
        )
        
        if result.matched_count == 0:
            return jsonify({"error": "Produk tidak ditemukan"}), 404
            
        product = db.products.find_one({"_id": ObjectId(product_id)})
        return jsonify(parse_product(product))
        
    except Exception as e:
        return jsonify({"error": f"Error updating product: {str(e)}"}), 500

# DELETE: Hapus produk
@app.route('/api/products/<product_id>', methods=['DELETE'])
def delete_product(product_id):
    try:
        result = db.products.delete_one({"_id": ObjectId(product_id)})
        if result.deleted_count == 0:
            return jsonify({"error": "Produk tidak ditemukan"}), 404
        return jsonify({"message": "Produk berhasil dihapus"})
    except Exception as e:
        return jsonify({"error": f"Error deleting product: {str(e)}"}), 500

# ADMIN MANAGEMENT ENDPOINTS

# GET: Ambil semua admin
@app.route('/api/admins', methods=['GET'])
def get_admins():
    try:
        admins = db.admins.find({}, {'password': 0})  # Exclude password field
        admin_list = []
        for admin in admins:
            admin_list.append({
                '_id': str(admin['_id']),
                'username': admin['username']
            })
        return jsonify(admin_list)
    except Exception as e:
        return jsonify({'error': f'Error fetching admins: {str(e)}'}), 500

# CREATE: Tambah admin baru
@app.route('/api/admins', methods=['POST'])
def create_admin():
    try:
        data = request.json
        if not data or not data.get('username') or not data.get('password'):
            return jsonify({'error': 'Username dan password wajib diisi'}), 400
            
        username = data.get('username')
        password = data.get('password')
        
        if db.admins.find_one({'username': username}):
            return jsonify({'error': 'Username sudah ada'}), 400
            
        hashed_pw = generate_password_hash(password)
        result = db.admins.insert_one({'username': username, 'password': hashed_pw})
        
        return jsonify({
            '_id': str(result.inserted_id),
            'username': username,
            'message': 'Admin created successfully'
        }), 201
        
    except Exception as e:
        return jsonify({'error': f'Error creating admin: {str(e)}'}), 500

# DELETE: Hapus admin
@app.route('/api/admins/<admin_id>', methods=['DELETE'])
def delete_admin(admin_id):
    try:
        result = db.admins.delete_one({"_id": ObjectId(admin_id)})
        if result.deleted_count == 0:
            return jsonify({"error": "Admin tidak ditemukan"}), 404
        return jsonify({"message": "Admin berhasil dihapus"})
    except Exception as e:
        return jsonify({"error": f"Error deleting admin: {str(e)}"}), 500

# AUTHENTICATION ENDPOINTS

# Register admin (alias untuk create admin)
@app.route('/api/register', methods=['POST'])
def register_admin():
    return create_admin()

# GET: Info login endpoint
@app.route('/api/login', methods=['GET'])
def login_info():
    return jsonify({'message': 'Login endpoint ready', 'methods': ['POST']})

# POST: Login admin
@app.route('/api/login', methods=['POST'])
def login_admin():
    try:
        data = request.json
        if not data or not data.get('username') or not data.get('password'):
            return jsonify({'error': 'Username dan password wajib diisi'}), 400
            
        username = data.get('username')
        password = data.get('password')
        admin = db.admins.find_one({'username': username})
        
        if not admin or not check_password_hash(admin['password'], password):
            return jsonify({'error': 'Username/password salah'}), 401
            
        return jsonify({'message': 'Login berhasil', 'admin': {'username': username}}), 200
        
    except Exception as e:
        return jsonify({'error': f'Login error: {str(e)}'}), 500

# STATISTICS ENDPOINT
@app.route('/api/stats')
def get_stats():
    try:
        total_products = db.products.count_documents({})
        total_admins = db.admins.count_documents({})
        return jsonify({
            'total_products': total_products,
            'total_admins': total_admins,
            'status': 'ok'
        })
    except Exception as e:
        return jsonify({'error': f'Stats error: {str(e)}'}), 500

if __name__ == '__main__':
    print("🚀 Flask Backend Starting...")
    print(f"📊 Database: {MONGO_URI}")
    print(f"📁 Upload folder: {UPLOAD_FOLDER}")
    print("🌐 CORS enabled for frontend")
    print("📋 Available endpoints:")
    print("   GET  /api/health")
    print("   GET  /api/products")
    print("   POST /api/products")
    print("   GET  /api/products/<id>")
    print("   PUT  /api/products/<id>")
    print("   DELETE /api/products/<id>")
    print("   GET  /api/admins")
    print("   POST /api/admins")
    print("   DELETE /api/admins/<id>")
    print("   GET  /api/login")
    print("   POST /api/login")
    print("   POST /api/register")
    print("   POST /api/upload")
    print("   GET  /api/stats")
    app.run(debug=True, host='0.0.0.0', port=5000)
