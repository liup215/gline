import subprocess
import sys
import time

p = subprocess.Popen(
    [sys.executable, "-m", "http.server", "8888"],
    cwd=r"C:\Users\22569\Workspace\gline\demo-spa",
    stdout=open(r"C:\Users\22569\Workspace\gline\server.log", "w"),
    stderr=open(r"C:\Users\22569\Workspace\gline\server.err", "w")
)
print("PID:", p.pid)
time.sleep(1)
print("Server should be running on port 8888")
