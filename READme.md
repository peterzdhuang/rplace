

Frontend : 

- Canvas of 1000 x 1000
- Each of the pixels can be (16 colours)
- username input page
- frontend will calculate what needs to be rerendered

Backend :

We want concurency,

if users joins 
connect to server
server give back the board state (list of pixels)
whenever a user makes a change 
this change broadcasted to everyone


- pixel 
    - username
    - color (r,g, b)
    - position (x,y)
 
- board 
    - size (m by n)

- user 
    - username 
    - token for cookie (to enforce a 5 minute timer between places)
    - ip tracking (make sure its not at ip address)
- We want the user's username, like who placed it 
