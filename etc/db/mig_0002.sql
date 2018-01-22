-- Add table column to beacons
alter table beacon_list
  -- 70dbm is my default power for ibeacons
  add column txpower integer not null default -70;

alter table beacon_list
  rename to ibeacons;

