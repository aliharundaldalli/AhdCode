/**
 * Modern Hero Carousel / Slider Controller
 * Supports autoplay, pause-on-hover, dots, prev/next arrows, and touch swiping.
 */
document.addEventListener('DOMContentLoaded', () => {
  const slider = document.querySelector('.hero-slider-container');
  if (!slider) return;

  const slides = slider.querySelectorAll('.hero-slide');
  const indicators = slider.querySelectorAll('.hero-indicator');
  const prevBtn = slider.querySelector('.hero-slider-btn-prev');
  const nextBtn = slider.querySelector('.hero-slider-btn-next');
  const progressBar = slider.querySelector('.hero-slider-progress-bar');

  if (slides.length <= 1) {
    if (prevBtn) prevBtn.style.display = 'none';
    if (nextBtn) nextBtn.style.display = 'none';
    if (indicators.length) indicators[0].parentElement.style.display = 'none';
    return;
  }

  let currentIndex = 0;
  const slideCount = slides.length;
  const intervalTime = 6000; // 6 seconds per slide
  let slideTimer = null;
  let isPaused = false;

  function showSlide(index) {
    slides.forEach((slide, i) => {
      slide.classList.toggle('active', i === index);
    });

    indicators.forEach((dot, i) => {
      dot.classList.toggle('active', i === index);
      dot.setAttribute('aria-current', i === index ? 'true' : 'false');
    });

    currentIndex = index;
    restartProgressBar();
  }

  function nextSlide() {
    const nextIndex = (currentIndex + 1) % slideCount;
    showSlide(nextIndex);
  }

  function prevSlide() {
    const prevIndex = (currentIndex - 1 + slideCount) % slideCount;
    showSlide(prevIndex);
  }

  function restartProgressBar() {
    if (!progressBar) return;
    progressBar.style.transition = 'none';
    progressBar.style.width = '0%';
    setTimeout(() => {
      if (!isPaused) {
        progressBar.style.transition = `width ${intervalTime}ms linear`;
        progressBar.style.width = '100%';
      }
    }, 40);
  }

  function startAutoplay() {
    stopAutoplay();
    isPaused = false;
    restartProgressBar();
    slideTimer = setInterval(() => {
      if (!isPaused) {
        nextSlide();
      }
    }, intervalTime);
  }

  function stopAutoplay() {
    if (slideTimer) {
      clearInterval(slideTimer);
      slideTimer = null;
    }
  }

  function pauseAutoplay() {
    isPaused = true;
    if (progressBar) {
      const computedWidth = window.getComputedStyle(progressBar).width;
      progressBar.style.transition = 'none';
      progressBar.style.width = computedWidth;
    }
  }

  function resumeAutoplay() {
    isPaused = false;
    startAutoplay();
  }

  // Event Listeners
  if (nextBtn) {
    nextBtn.addEventListener('click', (e) => {
      e.preventDefault();
      nextSlide();
      startAutoplay();
    });
  }

  if (prevBtn) {
    prevBtn.addEventListener('click', (e) => {
      e.preventDefault();
      prevSlide();
      startAutoplay();
    });
  }

  indicators.forEach((dot, i) => {
    dot.addEventListener('click', (e) => {
      e.preventDefault();
      showSlide(i);
      startAutoplay();
    });
  });

  slider.addEventListener('mouseenter', pauseAutoplay);
  slider.addEventListener('mouseleave', resumeAutoplay);

  // Keyboard navigation
  slider.setAttribute('tabindex', '0');
  slider.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowRight') {
      nextSlide();
      startAutoplay();
    } else if (e.key === 'ArrowLeft') {
      prevSlide();
      startAutoplay();
    }
  });

  // Touch Swipe Support
  let touchStartX = 0;
  let touchEndX = 0;

  slider.addEventListener('touchstart', (e) => {
    touchStartX = e.changedTouches[0].screenX;
  }, { passive: true });

  slider.addEventListener('touchend', (e) => {
    touchEndX = e.changedTouches[0].screenX;
    const diff = touchEndX - touchStartX;
    if (Math.abs(diff) > 45) {
      if (diff < 0) {
        nextSlide();
      } else {
        prevSlide();
      }
      startAutoplay();
    }
  }, { passive: true });

  // Initial activation
  showSlide(0);
  startAutoplay();
});
