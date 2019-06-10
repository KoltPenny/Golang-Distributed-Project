from flask import Flask, render_template, request, url_for, json
import requests
app = Flask(__name__)


@app.route('/')
def index():
    #lista = glob.glob("/home/**/*.*")
    #return render_template('formulario.html', list=lista)
    return render_template('index.html')

@app.route('/process',methods=['GET', 'POST'])
def process():

    if request.method == 'POST':
        newd = request.get_json()
        
        r = requests.post('http://localhost:8080/report','',json=newd)
        

        #return jsonify({'error':'Missing data!'})
        return ""

if __name__ == '__main__':
    app.run(debug = True)
   
