#!/bin/bash

echo "************************************************************"
echo "Setting up users..."
echo "************************************************************"

# create root user
nohup gosu mongodb mongo admin --eval "db.createUser({user: 'root', pwd: 'root', roles:[{ role: 'root', db: 'admin' }]});"

# create app user/database
nohup gosu mongodb mongo admin --eval "db.createUser({ user: 'admin', pwd: 'admin', roles: ['userAdminAnyDatabase', 'dbAdminAnyDatabase', 'readWriteAnyDatabase']});"

echo "************************************************************"
echo "Shutting down"
echo "************************************************************"
nohup gosu mongodb mongo admin --eval "db.shutdownServer();"