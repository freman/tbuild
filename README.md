# tbuild (and twatch)

I'm using vmware fusion to run a linux binary for testing (using netfilter code, so yay).

Because I was using simple vmware shared folders to share the code between linux and osx none of the usual fsnotify based builders was working.

I smashed this out in 10 mins to do the job.

## Commands

### tbuild

* Args are passed to the built command
* Won't stop the working command until the build succeeds

```
	go install github.com/freman/tbuild/cmd/tbuild
	cd $dir
	tbuild arg arg arg
```

### twatch

```
	go install github.com/freman/tbuild/cmd/twatch
	cd $dir
	twatch -remote ip
```

##Todo

* Configuration