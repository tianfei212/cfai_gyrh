module.exports = {
  apps: [{
    name: "gyrh-app",
    script: "./server.js",
    env: {
      NODE_ENV: "production",
      PORT: 3400
    },
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '1G'
  }]
};
