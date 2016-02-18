# smlr

smlr waits for service dependencies.

smlr is short for "sommelier", AKA a waiter with special knowledge and training.

# Use

smlr can wait for several types of dependencies:

- http
- tcp
- script

Each of them has different use cases. In general, you should prefer to use HTTP
or TCP for service depdendencies, and only use script if you need to do
something out of the ordinary (but please do open an issue with your use case!)

You can use it in, for example, a systemd unit file:

```
ExecStartPre=smlr http http://your-service-dependency.cluster.local:1234
ExecStart=/usr/bin/your-service
```

In this configuration, smlr will wait for a while (the default is 5 minutes) for
the service to be available and then exit with a failure if it is not up. It
adds jitter to avoid overloading a newly started dependency service.
