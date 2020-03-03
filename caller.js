const { spawn, execFile } = require('child_process');


let subprocess = execFile('udpTest.exe', [
  "-l",
  "-o",
  "client"
], {
  // detached: true,
  stdio: ['ignore', 1, 2] // ['ignore', process.stdout, process.stderr]
})


subprocess.stdout.on("data", (data) => {
  // console.log(data )
})
subprocess.stderr.on("data", (err) => {
  console.log("xiix", err ,"22")
})

// subprocess.unref()
console.log('Bye')
// process.exit()