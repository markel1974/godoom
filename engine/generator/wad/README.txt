world.go genera i lati e lascia che il motore li gestisca.

Per riprodurre esattamente Doom nel sistema, dobbiamo:

Usare il BSP per generare poligoni convessi e matematicamente chiusi al 100%.

Sovrapporre i dati del WAD: se un lato poggia su un muro reale (One-Sided Linedef), diventa Kind: 2 (Muro).

Se un lato è un portale o è stato generato dal BSP per chiudere il poligono nel vuoto, deve essere marcato come aperto (Kind: 3).