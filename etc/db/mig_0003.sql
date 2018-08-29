create table webmap_configs (
  id serial primary key,
  title string not null,
  image blob not null,
  config jsonb not null
  -- Schema for jsonb described in webmap.go
);


create or replace function average_stamp_and_prev(moment timestamptz, lastx interval default '00:00:00.5'::interval) 
  returns table(beacon int, edge int, rssi numeric, distance real)
  as $$
  select beaconid, edgenodeid, avg(rssi) as arssi, 
      cast (power(10, (e.bias - avg(rssi))/(10 * e.gamma)) as real) as distance
  -- 10 ^ (bias - rssi / 10 * gamma)
  from beacon_log as l, edge_node as e
  where datetime < $1 and datetime > $1 - $2 
  and l.edgenodeid = e.id
  group by beaconid, edgenodeid, gamma, bias
  order by beaconid, edgenodeid; $$
language SQL;

create view edge_locations as 
    select id, a.a[1] as x, a.a[2] as y, a.a[3] as z 
        from (select id, 
              regexp_matches(location, 
              '\s*\((-?\d+(?:.\d+)?),\s*(-?\d+(?:.\d+)?),\s*(-?\d+(?:.\d+)?)\)\s*') as a 
              from edge_node) as a order by id;
