create table if not exists users (
  id           bigserial primary key,
  login        text not null unique,
  pass_hash    bytea not null,
  first_name   text,
  last_name    text,
  birth_date   date,
  email        text,
  phone        text
);