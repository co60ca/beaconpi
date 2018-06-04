#!/bin/bash
set -eu

source environment.cfg
source sharedfun
dbdir="../db"

if ! psql -U "${DBUSER}" "${DBNAME}" -c "select 1" > /dev/null; then
  info "DB didn't exist, creating"
  createdb -U "${DBUSER}" "${DBNAME}" || iffail "Failed to create DB"
fi

# Check migration version
migration=$(echo "select level from migration_level" | psql -U "${DBUSER}" -t "${DBNAME}" 2> /dev/null)

# Check if anything in migration
if ! [ "${migration}" ] ; then
  psql -U "${DBUSER}" "${DBNAME}" < "${dbdir}/mig_0000.sql"
  migration=0
fi

info "Migration at ${migration}"

for fname in "${dbdir}/"*.sql ; do
  match="${dbdir}/mig_([0-9]{4}).sql"
  [[ ${fname} =~ $match ]]
  num=${BASH_REMATCH[1]:-""}

  [ "${num}" ] || iffail "Failed to match"

  if [ ${migration} -lt ${num} ] ; then
    psql -U "${DBUSER}" "${DBNAME}" < $fname > /dev/null
    info "Completed ${fname}"
  else 
    info "Skip ${fname}"
  fi
done

echo "update migration_level set level = ${num}" | psql -U "${DBUSER}" "${DBNAME}" > /dev/null
