#!/bin/bash

# builds and run the airline example

rm airline.o
go run cmd/atomic/main.go --verbose example/airline.atomic
gcc link/atomic.c airline.o -o airline
objdump -d airline.o
./airline
