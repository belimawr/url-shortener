CREATE TABLE urls (
       id SERIAL PRIMARY KEY,
       token varchar(50) UNIQUE NOT NULL,
       url varchar(2048) NOT NULL -- https://stackoverflow.com/questions/417142/what-is-the-maximum-length-of-a-url-in-different-browsers
);
