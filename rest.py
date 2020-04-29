import hashlib
import hmac
import base64

def make_digest(message, key):
    
    key = bytes(key, 'UTF-8')
    message = bytes(message, 'UTF-8')
    
    digester = hmac.new(key, message, hashlib.sha1)
    signature1 = digester.digest()
    signature2 = base64.urlsafe_b64encode(signature1)
    
    return str(signature2, 'UTF-8')

def stringToBase64(s):
    return base64.b64encode(s.encode('utf-8'))

result = make_digest(f'1595934095844:user', 'PJC')
print("username", "1595934095844:hi")
print("password", stringToBase64(result))
