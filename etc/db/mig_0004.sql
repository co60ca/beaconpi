create table system_errors (
  id serial primary key,
  datetime timestamp with time zone default current_timestamp, 
  error_id integer default null,
  error_level integer,
  error_text text not null,
  countn integer not null default 1,
  edgenodeid integer references edge_node default null
);

comment on column system_errors.error_level is '0: Trace, 1: Debug, 2: Info, 3: Warn, 4: Error, 5: Fatal';


alter table ibeacons add column enabled boolean not null default true;
alter table edge_node add column enabled boolean not null default true;

CREATE OR REPLACE FUNCTION inactive_beacons(lastx interval DEFAULT '00:10:00'::interval)
         RETURNS TABLE(label character varying, uuid uuid, major integer, minor integer)
         LANGUAGE plpgsql
       AS $function$
           BEGIN
             return query select a.label, a.uuid, a.major, a.minor
               from ibeacons as a
             left join
               beacon_log as b
               on a.id = b.beaconid
               and b.datetime > current_timestamp - $1 where datetime is null
               and a.enabled = true;
           END; $function$

CREATE OR REPLACE FUNCTION inactive_edges(lastx interval DEFAULT '00:01:00'::interval)
 RETURNS TABLE(id integer, title character varying, room text, location text, description text)
 LANGUAGE sql
AS $function$
    select id, title, room, location, description
    from edge_node
    where lastupdate < current_timestamp - $1 
    and enabled = true $function$
