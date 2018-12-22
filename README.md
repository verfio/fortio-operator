## Overview

This project is based on the [Operator Framework][of-home], an open source toolkit to manage Kubernetes native applications, called Operators, in an effective, automated, and scalable way. Read more in the [introduction blog post][of-blog].

[Fortio][fortio-home] is a load testing tool.
Fortio runs at a specified query per second (qps) and records an histogram of execution time and calculates percentiles (e.g. p99 ie the response time such as 99% of the requests take less than that number (in seconds, SI unit)). It can run for a set duration, for a fixed number of calls, or until interrupted (at a constant target QPS, or max speed/load per connection/thread).

The name fortio comes from greek φορτίο which means load/burden.

## Installation


[of-home]: https://github.com/operator-framework
[of-blog]: https://coreos.com/blog/introducing-operator-framework
[fortio-home]: https://github.com/fortio/fortio