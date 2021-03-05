# httping

httping is a small program to request http server in order to print statuscode and the time to get the response.

Example :

```
$ httping -u https://www.github.com -s 500
connected to https://www.github.com, seq=1 time=761.159 bytes=206779 StatusCode=200
connected to https://www.github.com, seq=2 time=147.326 bytes=206779 StatusCode=200
connected to https://www.github.com, seq=3 time=143.971 bytes=206779 StatusCode=200
connected to https://www.github.com, seq=4 time=138.060 bytes=206779 StatusCode=200
^Csignal: interrupt
```


# build

```
go build . 
```

# Usage

```
httping:
  -s int
        time to sleep between two tries. (default: 200) (default 200)
  -u string
        url to "ping"
exit status 2
```

