const $ = selector => document.querySelector(selector);
const authView=$('#auth-view'),chatView=$('#chat-view'),authForm=$('#auth-form'),authError=$('#auth-error');
const loginTab=$('#login-tab'),registerTab=$('#register-tab'),authSubmit=$('#auth-submit');
const messages=$('#messages'),chatForm=$('#chat-form'),input=$('#question'),send=$('#send');
let authMode='login';
const sessionKey='financial-agent-session';
let sessionId=localStorage.getItem(sessionKey);
if(!sessionId){sessionId=crypto.randomUUID?crypto.randomUUID():`web-${Date.now()}-${Math.random()}`;localStorage.setItem(sessionKey,sessionId)}

function setMode(mode){authMode=mode;const login=mode==='login';loginTab.classList.toggle('active',login);registerTab.classList.toggle('active',!login);authSubmit.textContent=login?'登录':'注册';$('#password').autocomplete=login?'current-password':'new-password';authError.textContent=''}
loginTab.addEventListener('click',()=>setMode('login'));registerTab.addEventListener('click',()=>setMode('register'));
function showChat(user){$('#current-user').textContent=user.username;authView.classList.add('hidden');chatView.classList.remove('hidden');input.focus()}
function showAuth(){chatView.classList.add('hidden');authView.classList.remove('hidden')}
async function api(path,options={}){const response=await fetch(path,{...options,headers:{'Content-Type':'application/json',...(options.headers||{})}});let data={};if(response.status!==204)data=await response.json().catch(()=>({}));if(!response.ok)throw new Error(data.error||'请求失败');return data}

authForm.addEventListener('submit',async event=>{event.preventDefault();authError.textContent='';authSubmit.disabled=true;try{const user=await api(`/api/auth/${authMode}`,{method:'POST',body:JSON.stringify({username:$('#username').value.trim(),password:$('#password').value})});showChat(user)}catch(error){authError.textContent=error.message}finally{authSubmit.disabled=false}});
$('#logout').addEventListener('click',async()=>{await api('/api/auth/logout',{method:'POST'}).catch(()=>{});showAuth()});

function appendMessage(role,text,extraClass=''){const item=document.createElement('div');item.className=`message ${role} ${extraClass}`;if(role==='assistant'){const avatar=document.createElement('div');avatar.className='avatar';avatar.textContent='安';item.appendChild(avatar)}const bubble=document.createElement('div');bubble.className='bubble';bubble.textContent=text;item.appendChild(bubble);messages.appendChild(item);messages.scrollTop=messages.scrollHeight;return item}
function resizeInput(){input.style.height='auto';input.style.height=`${Math.min(input.scrollHeight,130)}px`}
input.addEventListener('input',resizeInput);input.addEventListener('keydown',event=>{if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();chatForm.requestSubmit()}});
chatForm.addEventListener('submit',async event=>{event.preventDefault();const question=input.value.trim();if(!question||send.disabled)return;appendMessage('user',question);input.value='';resizeInput();send.disabled=true;const typing=appendMessage('assistant','正在为您查询…','typing');try{const result=await api('/api/chat',{method:'POST',body:JSON.stringify({session_id:sessionId,question})});typing.remove();appendMessage('assistant',result.answer)}catch(error){typing.remove();appendMessage('assistant',`抱歉，服务暂时不可用：${error.message}`);if(error.message==='请先登录')showAuth()}finally{send.disabled=false;input.focus()}});

api('/api/auth/me').then(showChat).catch(showAuth);
