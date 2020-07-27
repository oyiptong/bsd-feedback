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
