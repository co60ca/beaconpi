create or replace function active_beacons(lastx interval default '00:10:00'::interval) returns table(label text, uuid uuid, major int, minor int)
  as $$
    select distinct label, uuid, major, minor
    from ibeacons as a, beacon_log as b
    where b.beaconid = a.id and
    datetime > current_timestamp - $1 $$
language SQL;

create or replace function inactive_beacons(lastx interval default '00:10:00'::interval) returns table(label text, uuid uuid, major int, minor int)
  as $$
    select distinct label, uuid, major, minor
    from ibeacons as a, beacon_log as b
    where b.beaconid != a.id and
    datetime > current_timestamp - $1 $$
language SQL;

create or replace function inactive_edges(lastx interval default '00:01:00'::interval) 
  returns table(title varchar(60), room text, location text, description text)
  as $$
    select title, room, location, description
    from edge_node
    where lastupdate < current_timestamp - $1 $$
language SQL;
