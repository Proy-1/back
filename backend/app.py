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
# Configure CORS more explicitly
CORS(app, origins=["http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:3000", "http://127.0.0.1:8080"])

# MongoDB connection
MONGO_URI = os.getenv('MONGO_URI', 'mongodb://localhost:27017/pitipaw')
client = MongoClient(MONGO_URI)
db = client.get_default_database()

UPLOAD_FOLDER = 'static/uploads'
ALLOWED_EXTENSIONS = {'png', 'jpg', 'jpeg', 'gif'}
app.config['UPLOAD_FOLDER'] = UPLOAD_FOLDER

if not os.path.exists(UPLOAD_FOLDER):
    os.makedirs(UPLOAD_FOLDER)

def allowed_file(filename):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS

@app.route('/api/health')
def health_check():
    try:
        # Test database connection
        db.products.count_documents({})
        return jsonify({'status': 'ok', 'message': 'Backend is running', 'database': 'connected'})
    except Exception as e:
        return jsonify({'status': 'error', 'message': 'Database connection failed', 'error': str(e)}), 500

# CREATE: Tambah produk baru
def parse_product(product):
    return {
        "_id": str(product.get("_id")),
        "name": product.get("name"),
        "price": product.get("price"),
        "description": product.get("description"),
        "image_url": product.get("image_url"),
    }

@app.route('/api/upload', methods=['POST'])
def upload_image():
    if 'image' not in request.files:
        return jsonify({'error': 'No file part'}), 400
    file = request.files['image']
    if file.filename == '':
        return jsonify({'error': 'No selected file'}), 400
    if file and allowed_file(file.filename):
        filename = secure_filename(file.filename)
        filepath = os.path.join(app.config['UPLOAD_FOLDER'], filename)
        file.save(filepath)
        url = f'/static/uploads/{filename}'
        return jsonify({'image_url': url}), 201
    return jsonify({'error': 'File not allowed'}), 400

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

@app.route('/api/register', methods=['POST'])
def register_admin():
    try:
        data = request.json
        if not data or not data.get('username') or not data.get('password'):
            return jsonify({'error': 'Username dan password wajib diisi'}), 400
            
        username = data.get('username')
        password = data.get('password')
        
        if db.admins.find_one({'username': username}):
            return jsonify({'error': 'Username sudah ada'}), 400
            
        hashed_pw = generate_password_hash(password)
        db.admins.insert_one({'username': username, 'password': hashed_pw})
        return jsonify({'message': 'Admin registered'}), 201
        
    except Exception as e:
        return jsonify({'error': f'Registration error: {str(e)}'}), 500

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

# Serve static files
@app.route('/static/uploads/<path:filename>')
def serve_static(filename):
    return send_from_directory(app.config['UPLOAD_FOLDER'], filename)

# Endpoint untuk mendapatkan statistik (opsional)
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
    app.run(debug=True, host='0.0.0.0', port=5000)
