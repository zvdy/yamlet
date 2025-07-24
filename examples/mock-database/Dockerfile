# Simple mock database service
FROM node:18-alpine

WORKDIR /app

# Create a simple package.json
RUN echo '{"name":"mock-database","version":"1.0.0","main":"server.js","dependencies":{"express":"^4.18.0"}}' > package.json

# Install dependencies
RUN npm install

# Copy server code
COPY server.js .

EXPOSE 3306

CMD ["node", "server.js"]
