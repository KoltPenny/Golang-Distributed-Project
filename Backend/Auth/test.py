from flask import Flask, render_template, request
app = Flask(__name__)


@app.route('/Update',methods=['POST'])
def process():
    path = request.get_json()
    
    if 'usuario' == path['Table_name']:
        return "OK"
    else: return "FAILED"

if __name__ == '__main__':
   app.run(debug = True)
