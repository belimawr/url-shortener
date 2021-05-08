# URL Shortener

This repository contains the code for a URL shortener that I'm
developing as part of my live-streams on
[YouTube](https://www.youtube.com/channel/UCjrsXc8nA4tY7j3uk5G-jLQ/featured). As
of now this project is far from complete, but at some point it will be
"production-ready".

## Database
We use Postgres as a database, to start it for local devlopment using
Docker, run:
```
docker run --name url-shortener-postgres -it -e POSTGRES_HOST_AUTH_METHOD=trust  -e POSTGRES_DB=url -e POSTGRES_USER=db_user -p 5432:5432  postgres:alpine
```

To connect to the database using docker and `psql`, run:
```
docker exec -it url-shortener-postgres psql -U db_user url
```

To create the tables use the following SQL statement:
```sql
CREATE TABLE urls (
       id SERIAL PRIMARY KEY,
       token varchar(50) UNIQUE NOT NULL,
       url varchar(2048) UNIQUE NOT NULL
);
```

# LICENCE
GPLv3
