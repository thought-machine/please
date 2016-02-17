#! /bin/bash

BASE=$(basename $0)
NAME=${BASE//-/_}
BIN=/usr/bin/${NAME}_server

# TODO(pebers): Make this stuff configurable sensibly
export VERBOSITY=2
export PORT=7677
export DIR=/var/cache/plz_rpc
export LOW_WATER_MARK=45G
export HIGH_WATER_MARK=50G
export CLEAN_FREQUENCY=10

PIDFILE=/var/run/${NAME}.pid

start() {
  echo "starting plz cache"
  start-stop-daemon --make-pidfile --pidfile $PIDFILE --start --background --startas /bin/bash -- -c "$BIN -v $VERBOSITY -p $PORT -d $DIR -l $LOW_WATER_MARK -i $HIGH_WATER_MARK -f $CLEAN_FREQUENCY > /var/log/${NAME}.log 2>&1"
}

stop() {
  echo "stopping plz cache"
  start-stop-daemon --pidfile $PIDFILE --stop $BIN
}

status() {
  ps aux | grep $BIN | grep -v grep > /dev/null 2>&1
  out=$?
  if [ $out -eq 0 ]
  then
    echo 'Running!'
  else
    echo 'Not Running!'
  fi
  exit $out
}

# Carry out specific functions when asked to by the system
case "$1" in
  start)
    start
    RETVAL=$?
    ;;
  stop)
    stop
    RETVAL=$?
    ;;
  restart)
    start
    stop
    RETVAL=$?
    ;;
  status)
    status
    RETVAL=$?
    ;;
  *)
    echo "Usage: $0 {start|stop|restart}"
    exit 1
    ;;
esac

exit $RETVAL
