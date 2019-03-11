create table system_errors (
  id serial primary key,
  datetime timestamp with time zone default current_timestamp, 
  error_id integer default null,
  error_level integer,
  error_text text not null,
  edgenodeid integer references edge_node default null
);

comment on column system_errors.error_level is '0: Trace, 1: Debug, 2: Info, 3: Warn, 4: Error, 5: Fatal';

