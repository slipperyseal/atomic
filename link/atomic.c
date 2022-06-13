
#include "stdio.h"

int makeBoardingPass(void*,void*,void*,void*,void*);

int main(int argc, char *argv[]) {
    char* passenger[] = { "Slippery Seal" };
    char* airport[] = { "Adelaide", "ADL"};
    char* gate[] = { "Gate 1" };
    char* flight[] = { "AA001" };
    char* boardingPass[4];

    makeBoardingPass(&passenger, &airport, &gate, &flight, &boardingPass);

    printf("\n\nmakeBoardingPass()\n\n  passengerName: %s\n  airportCode: %s\n  flightNumber: %s\n  gateNumber: %s\n\n",
        boardingPass[0], boardingPass[1], boardingPass[2], boardingPass[3]);
}
