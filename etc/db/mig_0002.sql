create or replace function active_beacons(lastx interval default '00:10:00'::interval) returns table(label varchar(40), uuid uuid, major int, minor int)
  as $$
  BEGIN
    return query select distinct a.label, a.uuid, a.major, a.minor
    from ibeacons as a, beacon_log as b
    where b.beaconid = a.id and
    datetime > current_timestamp - $1; 
  END; $$ language plpgsql;

create or replace function inactive_beacons(lastx interval default '00:10:00'::interval) returns table(label varchar(40), uuid uuid, major int, minor int)
  as $$
    BEGIN
      return query select a.label, a.uuid, a.major, a.minor
        from ibeacons as a
      left join
        beacon_log as b
        on a.id = b.beaconid
        and b.datetime > current_timestamp - $1 where datetime is null;
    END; $$ language plpgsql;

create or replace function inactive_edges(lastx interval default '00:01:00'::interval) 
  returns table(title varchar(60), room text, location text, description text)
  as $$
    select title, room, location, description
    from edge_node
    where lastupdate < current_timestamp - $1 $$
language SQL;
