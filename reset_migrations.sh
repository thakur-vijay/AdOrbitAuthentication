
#!/bin/bash

set -e

echo "⚠️ Resetting AdOrbit Auth Database..."

docker exec -i adorbit-auth-postgres \
  psql -v ON_ERROR_STOP=1 -U adorbit_auth -d adorbit_auth <<'SQL'

DROP SCHEMA public CASCADE;

CREATE SCHEMA public AUTHORIZATION adorbit_auth;

SQL

echo "✅ Database completely reset!"
echo "🚀 Ready to run fresh migrations."
