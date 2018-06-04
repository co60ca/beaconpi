#!/bin/bash

source environment.cfg

"${BINDIR}metricsserv" "--db-driver-name=postgres" \
  --db-datasource-name="user=${DBUSER} dbname=${DBNAME} sslmode=disable" \
  --port=${METRICSPORT}
