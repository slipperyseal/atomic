#!/bin/bash

# builds and run the airline example

rm airline.o
go run cmd/atomic/main.go --verbose example/airline.atomic
objdump -d airline.o
gcc link/atomic.c airline.o -o airline
./airline
