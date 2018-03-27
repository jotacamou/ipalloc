ipalloc
=======

IP address allocator and scanner for xCAT.

Scanner:
- Talks to the xCAT RESTful API to retrieve the networks to be scanned.
- Scans for responding IP addresses and reverse DNS resolution (PTR) records.
- Stores scan results in Mongo DB for persistance.

Microservice:
- Provides a rest endpoint for reserving the next available IP address on a given network.
- Provides a rest endpoint for releasing an IP address and mark it as available.
