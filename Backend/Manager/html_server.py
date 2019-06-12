from flask import Flask, render_template, request, url_for, json
#from flask_cors import CORS
import requests

app = Flask(__name__)
#CORS(app)

@app.route('/')
def index():
    #lista = glob.glob("/home/**/*.*")
    #return render_template('formulario.html', list=lista)
    return render_template('index.html')

@app.route('/getmap',methods=['GET', 'POST'])
def getmap():

    if request.method == 'GET':
        
        try:
            r = requests.get('http://localhost:8081/startMap')
        except requests.ConnectionError:
            print("Connection error. Aborting message...")
            print("Notifying user...")
            return "CONN-ERR"

        response = r.content.decode("utf-8")
        if sc(response) == "":
            return json.jsonify(r.json())
        print(sc(response))
        return sc(response)

#SWITCH-CASE FOR ERRORS
def sc(x):
    return {
        "RP-TO" : "CONN-TO",
        "RP-OK" : "OK",
        "RP-QU" : "QUERY"
    }.get(x,"")

if __name__ == '__main__':
    app.run(debug = True)
   
