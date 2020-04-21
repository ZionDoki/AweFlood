const { execFile, spawn } = require('child_process');
const os  = require('os');

const udpTester = os.platform() == 'win32' ? '/udpTest.exe' : '/AweFlood'

/**
 * start syn flood
 * @param {function} callback (null) callback fucntion when done
 * @param {function} errCallbackd (string err) error callback fucntion when error 
 * @param {object} options, {mode, targetPort, targetIP, duration, speed, log}
 */
function startUDPMission(callback, errCallback, options={}) {

  const {
    mode,
    targetPort,
    targetIP,
    duration,
    speed,
    log,
    special
  } = options;
  
  if (mode === "client") {
    console.log(`\n Start udp test as ${mode}`);
    if (targetIP && targetPort && speed && duration) {
      let subprocess = execFile(__dirname + udpTester, [
        "-o",
        mode,
        "-p",
        targetPort,
        "-i",
        targetIP,
        "-v",
        speed,
        "-d",
        duration,
        special ? "-s": "",
        log ? "-l" : ""
      ], {
        stdio: ['ignore', 1, 2] // ['ignore', process.stdout, process.stderr]
      })

      subprocess.stdout.on("data", (data) => {
        if (data.indexOf("Error") !== -1) {
          errCallback("Expired")
        } else {
          log ? console.log("[CLIENT INFO]: ", data) : ''
        }
      });

      subprocess.on("error", (error) => {
        errCallback(error);
        log ? console.log("[CLIENT ERR]: ", error) : ''
      });

      subprocess.on("exit", (msg) => {
        if (msg != 0) {
          errCallback()
        } else {
          callback()
        }
        log ? console.log("[CLIENT EXIT]: ", msg) : ''
      });

    } else {
      console.log(`\n[UDP Download ERR]: params err!`);
    }
  } else if (mode === "server") {
    console.log(`\n Start udp test as ${mode}`);
    if (targetIP && targetPort) {
      let subprocess = execFile(__dirname + udpTester, [
        "-o",
        mode,
        "-p",
        targetPort,
        special ? "-s": "",
        log ? "-l" : ""
      ], {
        stdio: ['ignore', 1, 2] // ['ignore', process.stdout, process.stderr]
      });

      subprocess.stdout.on("data", (data) => {
        if (data.indexOf("Error") !== -1) {
          errCallback("Expired")
        } else {
          log ? console.log("[SERVER INFO]: ", data) : ''
        }
      });

      subprocess.on("error", (error) => {
        errCallback(error);
        log ? console.log("[SERVER ERR]: ", error) : ''
      });

      subprocess.on("exit", (msg) => {
        if (msg != 0) {
          errCallback(msg)
        } else {
          callback()
        }
        log ? console.log("[SERVER EXIT]: ", msg) : ''
      });

    } else {
      errCallback(`params incomplete!`);
    }
  } else {
    errCallback(`params incorrect!`);
  }
}

module.exports = {
  startUDPMission
}
