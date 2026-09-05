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
  directLink.href = fileUrl;
  downloadLink.href = fileUrl;
  downloadLink.setAttribute('download', filename || 'cozum');

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
