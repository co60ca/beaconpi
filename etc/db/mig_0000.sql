create table migration_level (
  id serial primary key,
  level integer
);

insert into migration_level (level) values (0);

-- Table should gets augmented into the auth db
create table admin_table (
  id serial primary key,
  displayname text,
  email text,
  password bytea
);

