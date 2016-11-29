#!/bin/sh

kill `ps | grep gnatsd | cut -c 1-6`

