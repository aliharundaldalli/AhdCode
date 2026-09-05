function openSolutionModal(id, title, mimeType, filename) {
  var modal = document.getElementById('solutionModal');
  var modalTitle = document.getElementById('solutionModalTitle');
  var modalSubtitle = document.getElementById('solutionModalSubtitle');
  var modalBody = document.getElementById('solutionModalBody');
  var directLink = document.getElementById('solutionModalDirectLink');
  var downloadLink = document.getElementById('solutionModalDownloadLink');

  if (!modal) return;

  var fileUrl = '/solution/file?id=' + id;
  modalTitle.textContent = title || 'Çözüm Görüntüleyici';
  modalSubtitle.textContent = filename || '';
  if (directLink) directLink.href = fileUrl;
  if (downloadLink) {
    downloadLink.href = fileUrl;
    downloadLink.setAttribute('download', filename || 'cozum');
  }

  modalBody.innerHTML = '<div class="text-center py-5"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Yükleniyor...</span></div></div>';
  modal.classList.remove('d-none');
  document.body.style.overflow = 'hidden';

  if (mimeType === 'application/pdf') {
    modalBody.innerHTML = '<iframe src="' + fileUrl + '" width="100%" height="600" style="border: none; border-radius: 6px; min-height: 500px;"></iframe>';
  } else {
    var img = document.createElement('img');
    img.src = fileUrl;
    img.alt = filename || 'Çözüm';
    img.className = 'img-fluid rounded d-block mx-auto shadow-sm';
    img.style.maxHeight = '72vh';
    img.style.objectFit = 'contain';
    img.onload = function() {
      modalBody.innerHTML = '';
      modalBody.appendChild(img);
    };
    img.onerror = function() {
      modalBody.innerHTML = '<div class="alert alert-warning py-4">Görsel yüklenemedi. <a href="' + fileUrl + '" target="_blank" class="alert-link">Doğrudan açmayı deneyin</a>.</div>';
    };
  }
}

function closeSolutionModal() {
  var modal = document.getElementById('solutionModal');
  if (modal) {
    modal.classList.add('d-none');
    var modalBody = document.getElementById('solutionModalBody');
    if (modalBody) modalBody.innerHTML = '';
  }
  document.body.style.overflow = '';
}

document.addEventListener('keydown', function(e) {
  if (e.key === 'Escape') closeSolutionModal();
});

function togglePasswordVisibility(inputId, btn) {
  var input = document.getElementById(inputId);
  if (!input) return;
  if (input.type === 'password') {
    input.type = 'text';
    if (btn) {
      btn.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16"><path d="M13.359 11.238C15.06 9.72 16 8 16 8s-3-5.5-8-5.5a7.028 7.028 0 0 0-2.79.588l.77.771A5.944 5.944 0 0 1 8 3.5c2.12 0 3.879 1.168 5.168 2.457A13.134 13.134 0 0 1 14.828 8c-.058.087-.122.183-.195.288-.335.48-.83 1.12-1.465 1.755-.165.165-.337.328-.517.486l.708.709z"/><path d="M11.297 9.176a3.5 3.5 0 0 0-4.474-4.474l.823.823a2.5 2.5 0 0 1 2.829 2.829l.822.822zm-2.943 1.299.822.822a3.5 3.5 0 0 1-4.474-4.474l.823.823a2.5 2.5 0 0 0 2.829 2.829z"/><path d="M3.35 5.47c-.18.16-.353.322-.518.487A13.134 13.134 0 0 0 1.172 8l.195.288c.335.48.83 1.12 1.465 1.755C4.121 11.332 5.881 12.5 8 12.5c.716 0 1.39-.133 2.02-.36l.77.772A7.029 7.029 0 0 1 8 13.5C3 13.5 0 8 0 8s.939-1.721 2.641-3.238l.708.709zm10.296 8.884-12-12 .708-.708 12 12-.708.708z"/></svg>';
      btn.setAttribute('title', 'Şifreyi Gizle');
      btn.setAttribute('aria-label', 'Şifreyi Gizle');
    }
  } else {
    input.type = 'password';
    if (btn) {
      btn.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16"><path d="M16 8s-3-5.5-8-5.5S0 8 0 8s3 5.5 8 5.5S16 8 16 8zM1.173 8a13.133 13.133 0 0 1 1.66-2.043C4.12 4.668 5.88 3.5 8 3.5c2.12 0 3.879 1.168 5.168 2.457A13.133 13.133 0 0 1 14.828 8c-.058.087-.122.183-.195.288-.335.48-.83 1.12-1.465 1.755C11.879 11.332 10.119 12.5 8 12.5c-2.12 0-3.879-1.168-5.168-2.457A13.134 13.134 0 0 1 1.172 8z"/><path d="M8 5.5a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5zM4.5 8a3.5 3.5 0 1 1 7 0 3.5 3.5 0 0 1-7 0z"/></svg>';
      btn.setAttribute('title', 'Şifreyi Göster');
      btn.setAttribute('aria-label', 'Şifreyi Göster');
    }
  }
}

function setColorPreset(hex) {
  var textInput = document.getElementById('settings-color');
  var picker = document.getElementById('header_color_picker');
  if (textInput) textInput.value = hex;
  if (picker) picker.value = hex;
}

function updateColorFromInput(hex) {
  var picker = document.getElementById('header_color_picker');
  if (picker && /^#[0-9A-Fa-f]{6}$/.test(hex)) {
    picker.value = hex;
  }
}
