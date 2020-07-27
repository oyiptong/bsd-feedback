"use strict";

function sendEmail(letterID, sendCount) {
  if (sendCount > 0) {
    if(!window.confirm("Looks like you tried to send this letter earlier. Try again?")) {
      return;
    }
  }
  let body = document.querySelector("#letter-preview").innerText.trim();
  body = encodeURIComponent(body);
  const subject = encodeURIComponent("Parent Concerns about Distance Learning");
  const href = `mailto:cmountbenites@burlingameschools.org?subject=${subject}&body=${body}`;
  fetch(`/letter/record-send/${letterID}`, {method: 'POST'}).finally(() => {
    document.location.href = href;
  });
}

window.addEventListener('DOMContentLoaded', (event) => {
  const fbContainer = document.querySelector('#fb-share');
  const iframe = document.createElement('iframe');
  const currentURL = encodeURIComponent(window.location.href);
  iframe.src = `https://www.facebook.com/plugins/share_button.php?href=${currentURL}&layout=button&size=small&width=67&height=20`;
  iframe.width = "67";
  iframe.height = "20";
  iframe.style = "border:none;overflow:hidden";
  iframe.scrolling = "no";
  iframe.frameborder = "0";
  iframe.allowTransparency = "true";
  iframe.allow = "encrypted-media";
  fbContainer.appendChild(iframe);
});
