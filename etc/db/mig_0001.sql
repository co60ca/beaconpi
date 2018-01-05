-- Tables
drop index if exists beacon_list_uuid_index;
drop table if exists beacon_list cascade;
create table beacon_list (
  id serial primary key,
  label varchar(40) not null,
  uuid uuid not null,
  major integer not null,
  minor integer not null
);
create unique index beacon_list_uuid_index on beacon_list(uuid);

drop index if exists edge_node_uuid_index;
drop table if exists edge_node cascade;
create table edge_node (
  id serial primary key,
  uuid uuid not null,
  title varchar(60) not null,
  room text not null,
  location text not null,
  description text
);
create unique index edge_node_uuid_index on edge_node(uuid);

drop index if exists beacon_log_datetime;
drop index if exists beacon_log_beaconid;
drop index if exists beacon_log_edgenodeid;
drop table if exists beacon_log cascade;
create table beacon_log (
  datetime timestamp not null default current_timestamp,
  beaconid integer not null references beacon_list,
  edgenodeid integer not null references edge_node,
  rssi integer not null
);
create index beacon_log_datetime on beacon_log(datetime);
create index beacon_log_beaconid on beacon_log(beaconid);
create index beacon_log_edgenodeid on beacon_log(edgenodeid);

drop index if exists control_commands_edgenodeid;
drop table if exists control_commands cascade;
create table control_commands (
  id serial primary key,
  datetime timestamp not null default current_timestamp,
  completed boolean not null default FALSE,
  edgenodeid integer not null references edge_node,
  data text
);
create index control_commands_edgenodeid on control_commands(edgenodeid);

drop index if exists control_log_edgenodeid;
drop index if exists control_log_controlid;
drop table if exists control_log cascade;
create table control_log (
  datetime timestamp not null default current_timestamp,
  edgenodeid integer not null references edge_node,
  controlid integer references control_commands,
  data text
);
create index control_log_edgenodeid on control_log(edgenodeid);
create index control_log_controlid on control_log(controlid);

-- Data generating functions
create or replace function fake_generate_beacons(count integer) returns void as $$
declare
  uuidgen varchar(36);
begin
  for i in 1..count loop
    uuidgen := uuid_in(md5(random()::text || now()::text)::cstring);
    uuidgen := substring(uuidgen, 0, 15) || '4' || substring(uuidgen, 16, char_length(uuidgen));
    insert into beacon_list (label, uuid, major, minor)
            values ('Label-' || md5(random()::text),
                    cast (uuidgen as uuid),
                    0,
                    0);
  end loop;
end;
$$ language plpgsql;

create or replace function fake_generate_edges(count integer) returns void as $$
declare
  uuidgen varchar(36);
begin
  for i in 1..count loop
    uuidgen := uuid_in(md5(random()::text || now()::text)::cstring);
    uuidgen := substring(uuidgen, 0, 15) || '4' || substring(uuidgen, 16, char_length(uuidgen));
    insert into edge_node (title, uuid, room, location)
            values ('Title-' || md5(random()::text),
                    cast (uuidgen as uuid),
                    'Room-' || md5(random()::text),
                    'Location-' || md5(random()::text)
                   );
  end loop;
end;
$$ language plpgsql;

create or replace function fake_generate_beacon_logs(count integer) returns void as $$
declare
  beacon record;
  node record;
  temprcount integer;
begin
  for beacon in select id from beacon_list loop
    for node in select id from edge_node loop
      temprcount := round(random()*count/2 + count*0.5);
      for i in 1..temprcount loop
        insert into beacon_log (datetime, beaconid, edgenodeid, rssi)
          values (
            timestamp '2000-01-01 00:00' + interval '1 seconds ' * floor(60*60*24*random()),
            beacon.id, node.id,
            floor(random()*40-70)
          );
      end loop;
    end loop;
  end loop;
end;
$$ language plpgsql;

create or replace function fake_data(count float) returns void as $$
begin
  perform fake_generate_beacons(cast(greatest(8*count, 1) as integer));
  perform fake_generate_edges(cast(greatest(6*count, 1) as integer));
  perform fake_generate_beacon_logs(cast(10000*count as integer));
end;
$$ language plpgsql;

