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
      btn.innerHTML = '<i class="fa-regular fa-eye-slash"></i>';
      btn.setAttribute('title', 'Şifreyi Gizle');
      btn.setAttribute('aria-label', 'Şifreyi Gizle');
    }
  } else {
    input.type = 'password';
    if (btn) {
      btn.innerHTML = '<i class="fa-regular fa-eye"></i>';
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
