#!/bin/sh

kill `ps | grep nats-client-sim | cut -c 1-6`

