# Wiener Netze Smart Meter Client

This is a module and executable to pull smartmeter data from the [Smart Meter Portal of Wiener Netze][portal].

[portal]: https://smartmeter-web.wienernetze.at/

It draws quite some inspiration from
https://github.com/platysma/vienna-smartmeter and
https://github.com/fleinze/vienna-smartmeter/ with the following differences:
- implemented in go, obviously
- ships with a example binary, to show how it can be used to collect data and stream them out as metrics in the influx line protocol
- only implements a subset of the APIs available

Other things to know:
- To my knowledge the APIs are completely undocumented. I checked, what the
  above mentioned projects did, and used my browser's developer tools to figure
  out what the client needs to send and receives.
  Therefore, this thing might break at any point in time.
- Every request this client makes, leaves a little message for the owners of
  the various APIs, asking them to please document their stuff; via the
  user-agent header in the request. Will this help and get the APIs publicly
  documented? Probably not. Does it hurt? Don't think so.
- The responses from the different APIs are ..... interesting. The client
  should hopefully abstract most of those away.
- In case the APIs change in any way, this client might break, or might return
  wrong or incomplete data. You have been warned.
- Only things I need are implemented. If you want other stuff, pull other data
  from the smart meter portal, feel free to open a issue or, better yet, a PR!
