# Heis


Kømodul:
Bestillinger utenfra lagres i EN kø som alle forholder seg til. Bestillinger innenfra legges til i unike køer, en kø for hver heis.
Køene med bestillinger innenfra er offentlige, og brukes av alle heisene til å vurdere valg av neste handling. Dersom en heis
skal sette av noen i 1. etasje, er det jo ikke noe poeng at en annen heis også kjører dit. Denne måten å organisere køen på sikrer
at en ordre ikke mistes som følge av at to hendelser skjer samtidig: heis 1 forlater 4. etasje idet en ordre til fjerde etasje
legges til. Et faremoment er da at alle andre tror at heis 1 tar ordren, mens heis 1 tror noen andre tar den. Dette unngås ved 
felles kø for bestillinger utenfra.

Har vurdert å kombinere modulene Floor Manager og Queue Manager. Da blir det bare en modul som styrer lys - praktisk.

Har også vurdert en egen modul for å beregne neste handling. Det er ikke intuitivt interface at kømodul bestemmer neste handling.
