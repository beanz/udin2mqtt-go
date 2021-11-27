function load() {
  let message_timeout = null;
  let messageElt = document.getElementById('messages')
  function setMessage(msg) {
    messageElt.innerHTML = msg;
    clearTimeout(message_timeout);
    message_timeout = setTimeout(function () {
      messageElt.innerHTML = ""
    }, 5000);
  }

  var x = document.getElementsByClassName("createRelay");
  var i;
  for (i = 0; i < x.length; i++) {
    x[i].addEventListener('click', (event) => {
      var form = event.currentTarget.parentElement
      var name = form.getElementsByTagName("input")[0].value
      var selects = form.getElementsByTagName("select")
      var openIdx = selects[0].selectedIndex
      var open = selects[0].options[openIdx].text
      var closeIdx = selects[1].selectedIndex
      var close = selects[1].options[closeIdx].text
      if (openIdx == closeIdx) {
          setMessage("Please select different open/close relays!")
          return;
      }
      var type = selects[2].selectedIndex
      var param = name + "," + type + "," + open + "," + close
      var xmlhttp = new XMLHttpRequest();
      xmlhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
          var resp = JSON.parse(this.responseText);
          setMessage(resp.message)
        }
      };
      xmlhttp.open("GET", "/api/create/" + param, true);
      xmlhttp.send()
    })
  }

  var x = document.getElementsByClassName("enableDisable");
  var i;
  for (i = 0; i < x.length; i++) {
    x[i].addEventListener('change', (event) => {
      var dev = event.currentTarget.getAttribute('x-device');
      var param = "false"
      var msg = "disabled"
      if (event.currentTarget.checked) {
        param = "true"
        msg = "enabled"
      }
      var xmlhttp = new XMLHttpRequest();
      xmlhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
          var resp = JSON.parse(this.responseText);
          setMessage(resp.message)
        }
      };
      xmlhttp.open("GET", "/api/" + dev +"/enable/" + param, true);
      xmlhttp.send()
    })
  }
}
