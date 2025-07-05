from flask import Flask, jsonify, request
from flask_cors import CORS
from pymongo import MongoClient
import os
from dotenv import load_dotenv
from bson.objectid import ObjectId
from werkzeug.utils import secure_filename

# Load environment variables from .env file
load_dotenv()

app = Flask(__name__)
CORS(app)

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
    return jsonify({'status': 'ok', 'message': 'Backend is running'})

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
    data = request.json
    result = db.products.insert_one({
        "name": data.get("name"),
        "price": data.get("price"),
        "description": data.get("description"),
        "image_url": data.get("image_url")
    })
    new_product = db.products.find_one({"_id": result.inserted_id})
    return jsonify(parse_product(new_product)), 201

# READ: Ambil semua produk
@app.route('/api/products', methods=['GET'])
def get_products():
    products = db.products.find()
    return jsonify([parse_product(p) for p in products])

# READ: Ambil satu produk berdasarkan id
@app.route('/api/products/<product_id>', methods=['GET'])
def get_product(product_id):
    product = db.products.find_one({"_id": ObjectId(product_id)})
    if not product:
        return jsonify({"error": "Produk tidak ditemukan"}), 404
    return jsonify(parse_product(product))

# UPDATE: Ubah data produk
@app.route('/api/products/<product_id>', methods=['PUT'])
def update_product(product_id):
    data = request.json
    update_data = {
        "name": data.get("name"),
        "price": data.get("price"),
        "description": data.get("description"),
        "image_url": data.get("image_url")
    }
    result = db.products.update_one(
        {"_id": ObjectId(product_id)},
        {"$set": update_data}
    )
    if result.matched_count == 0:
        return jsonify({"error": "Produk tidak ditemukan"}), 404
    product = db.products.find_one({"_id": ObjectId(product_id)})
    return jsonify(parse_product(product))

# DELETE: Hapus produk
@app.route('/api/products/<product_id>', methods=['DELETE'])
def delete_product(product_id):
    result = db.products.delete_one({"_id": ObjectId(product_id)})
    if result.deleted_count == 0:
        return jsonify({"error": "Produk tidak ditemukan"}), 404
    return jsonify({"message": "Produk berhasil dihapus"})

if __name__ == '__main__':
    app.run(debug=True)
