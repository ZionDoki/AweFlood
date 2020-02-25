const { spawn, execFile } = require('child_process');


let subprocess = execFile('udpTest.exe', [
  "-l"
], {
  // detached: true,
  stdio: ['ignore', 1, 2] // ['ignore', process.stdout, process.stderr]
})


subprocess.stdout.on("data", (data) => {
  console.log("xiix", data ,"22")
})

// subprocess.unref()
console.log('Bye')
// process.exit()