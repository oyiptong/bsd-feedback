"use strict";

function sendEmail(letterID) {
  let body = document.querySelector("#letter-preview").innerText.trim();
  body = encodeURIComponent(body);
  const subject = encodeURIComponent("Parent Concerns about Distance Learning");
  const href = `mailto:cmountbenites@burlingameschools.org?subject=${subject}&body=${body}`;
  fetch(`/letter/record-send/${letterID}`).finally(() => {
    document.location.href = href;
  });
}
