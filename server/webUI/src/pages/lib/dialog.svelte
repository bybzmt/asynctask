<script>
  import { fade } from "svelte/transition";

  export let isShow;

  let top, left, height, width;

  function reset() {
    top = "0px";
    left = "0px";
    height = "auto";
    width = "auto";
  }
  reset();

  function position(node) {
    if (window.innerWidth > node.clientWidth + 100) {
      left = (window.innerWidth - node.clientWidth) / 2 + "px";
    } else {
      width = window.innerWidth - 100 + "px";
      left = "50px";
    }

    if (window.innerHeight > node.clientHeight + 100) {
      top = (window.innerHeight - node.clientHeight) / 2 + "px";
    } else {
      height = window.innerHeight - 100 + "px";
      top = "50px";
    }
  }

  function initPosition(node) {
    position(node);

    let timer = setInterval(() => position(node), 200);

    return {
      destroy() {
        clearInterval(timer);
        reset();
      },
    };
  }
</script>

{#if isShow}
  <div class="fixed w-screen h-screen inset-0 bg-[#00000022]" transition:fade={{ delay: 10, duration: 100 }}>
    <div
      class="fixed"
      use:initPosition
      style:top
      style:left
      style:height
      style:width
    >
      <slot />
    </div>
  </div>
{/if}
