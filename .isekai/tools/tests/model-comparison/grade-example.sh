#!/bin/bash
# grade-example.sh -- correctness grading for THIS test's example task (the slime-db-role
# dropRole/hardcoded-database bug in terraform-provider-mongodb). Correctness checks are
# inherently task-specific -- if you swap in a different task/repo (see README.md), write
# your own grading logic following this shape: objective, automatable pass/fail checks, not
# a subjective read of the diff.
#
# Usage: ./grade-example.sh <trials-dir>
set -e
DIR="${1:?Usage: grade-example.sh <trials-dir>}"

for trial in "$DIR"/with-* "$DIR"/without-*; do
  label="$(basename "$trial")"
  f="$trial/mongodb/resource_db_role.go"
  echo "=== $label ==="
  if [ ! -f "$f" ]; then echo "  FILE MISSING"; continue; fi
  awk '/^func resourceDatabaseRoleUpdate/,/^}/' "$f" > /tmp/grade_upd_$$.txt
  grep -q "dropRole" /tmp/grade_upd_$$.txt && c1="PASS" || c1="FAIL"
  grep -q 'Database("admin")' /tmp/grade_upd_$$.txt && c2="FAIL (still hardcoded admin)" || c2="PASS"
  ( cd "$trial" && go build ./... ) > /tmp/grade_build_$$.log 2>&1 && c3="PASS" || c3="FAIL"
  echo "  1) dropRole used in Update:   $c1"
  echo "  2) no hardcoded admin:        $c2"
  echo "  3) compiles:                  $c3"
  [ "$c3" = "FAIL" ] && sed 's/^/     /' /tmp/grade_build_$$.log
  rm -f /tmp/grade_upd_$$.txt /tmp/grade_build_$$.log
done
