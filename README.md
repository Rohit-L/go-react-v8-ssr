# go-react-v8-ssr: Work-in-progress

Go server serving a react-based website using server-side rendering powered by V8.

---

This repo is forked from the awesome [go-starter-kit](https://github.com/olebedev/go-starter-kit).

I forked that project and am modifying to get a better understanding of how it works.  And use some of the tools I'm more familiar with.  There are many changes, but some of the major ones include:
* rework the JS engine to use [v8](https://github.com/augustoroman/v8) instead of [duktape](https://github.com/olebedev/go-duktape)
* simplify the server code to be an easier-to-understand example
* use [modd](https://github.com/cortesi/modd) instead of [on](https://github.com/olebedev/on) for running the server.
* use [go-rice](https://github.com/GeertJohan/go.rice) instead of [go-bindata](https://github.com/jteeuwen/go-bindata) for embedding the data.
* use [govendor](https://github.com/kardianos/govendor) rather than [srlt](https://github.com/olebedev/srlt)

## Installation

Make sure you have:
* [golang](https://golang.org/)
* [node.js](https://nodejs.org/) with [npm](https://www.npmjs.com/), only to build the application bundle at compile time
* [GNU make](https://www.gnu.org/software/make/)

#### Clone the repo
```bash
$ git clone git@github.com:augustoroman/go-react-v8-ssr.git $GOPATH/src/github.com/<username>/<project>
$ cd $GOPATH/src/github.com/<username>/<project>
```

#### Install some go-based utilities:
```bash
$ go get -u github.com/kardianos/govendor
$ go get -u github.com/GeertJohan/go.rice/rice
$ go get -u https://github.com/cortesi/modd
```

#### Install dependencies:
```bash
$ govendor sync
$ npm install
```

#### Build V8
See instructions at https://github.com/augustoroman/v8

#### Add v8 symlinks
vendor/github.com/augustoroman/v8/symlink.sh <V8-DIR>

## Run development

Start dev server:

```
$ modd
```

that's it. Open [http://localhost:5001/](http://localhost:5001/)(if you use default port) at your browser. Now you ready to start coding your awesome project.

## Build

Install dependencies and type `NODE_ENV=production make build`. This rule produces the production webpack build and that is embedded into the go server, then builds the server. You can find the result at `./bin/server`.


---

## Project structure

##### The server's entry point
```
$ tree server
server
├── main.go <-- main function declared here
├── react-v8.go
├── bindata.go <-- this file is gitignored, it will appear at compile time
└── data
    ├── static
    |   └── build <-- this dir is populated by webpack automatically
    └── templates
        └── react.html
```

The `./server/` is flat golang package.

##### The client's entry point

It's simple React application

```
$ tree client
client
├── actions.js
├── components
│   ├── app
│   │   ├── favicon.ico
│   │   ├── index.js
│   │   └── styles.css
│   ├── homepage
│   │   ├── index.js
│   │   └── styles.css
│   ├── not-found
│   │   ├── index.js
│   │   └── styles.css
│   └── usage
│       ├── index.js
│       └── styles.css
├── css
│   ├── funcs.js
│   ├── global.css
│   ├── index.js
│   └── vars.js
├── index.js <-- main function declared here
├── reducers.js
├── router
│   ├── index.js
│   ├── routes.js
│   └── toString.js
└── store.js
```

The client app will be compiled into `server/data/static/build/`.  Then it will be embedded into go package via _go-bindata_. After that the package will be compiled into binary.

**Convention**: javascript app should declare [_main_](https://github.com/augustoroman/go-react-v8-ssr/blob/master/client/index.js#L4) function right in the global namespace. It will used to render the app at the server side.

## License
MIT
