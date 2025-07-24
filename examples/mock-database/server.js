const express = require('express');
const app = express();
const port = process.env.PORT || 3306;

// Mock database data
const mockData = {
  users: [
    { id: 1, name: 'John Doe', email: 'john@example.com' },
    { id: 2, name: 'Jane Smith', email: 'jane@example.com' }
  ],
  products: [
    { id: 1, name: 'Widget A', price: 19.99 },
    { id: 2, name: 'Widget B', price: 29.99 }
  ]
};

app.use(express.json());

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'healthy', timestamp: new Date().toISOString() });
});

// Get all users
app.get('/users', (req, res) => {
  console.log('Database query: SELECT * FROM users');
  res.json(mockData.users);
});

// Get user by ID
app.get('/users/:id', (req, res) => {
  const userId = parseInt(req.params.id);
  const user = mockData.users.find(u => u.id === userId);
  console.log(`Database query: SELECT * FROM users WHERE id = ${userId}`);
  
  if (user) {
    res.json(user);
  } else {
    res.status(404).json({ error: 'User not found' });
  }
});

// Get all products
app.get('/products', (req, res) => {
  console.log('Database query: SELECT * FROM products');
  res.json(mockData.products);
});

// Get product by ID
app.get('/products/:id', (req, res) => {
  const productId = parseInt(req.params.id);
  const product = mockData.products.find(p => p.id === productId);
  console.log(`Database query: SELECT * FROM products WHERE id = ${productId}`);
  
  if (product) {
    res.json(product);
  } else {
    res.status(404).json({ error: 'Product not found' });
  }
});

// Database connection info endpoint
app.get('/info', (req, res) => {
  res.json({
    database: 'mock_database',
    version: '1.0.0',
    connection_count: Math.floor(Math.random() * 10) + 1,
    uptime: process.uptime()
  });
});

app.listen(port, () => {
  console.log(`Mock Database Server running on port ${port}`);
  console.log('Available endpoints:');
  console.log('  GET /health - Health check');
  console.log('  GET /users - Get all users');
  console.log('  GET /users/:id - Get user by ID');
  console.log('  GET /products - Get all products');
  console.log('  GET /products/:id - Get product by ID');
  console.log('  GET /info - Database info');
});
