<script>
        export let teachers = [];
        export let subjects = [];
        export let classes = [];
        export let rooms = [];
        export let t;
        export let s;
        export let c;
        export let r;
        export let editing;
        export let addTeacher;
        export let addSubject;
        export let addClass;
        export let addRoom;
        export let saveEdit;
        export let cancelEdit;
        export let startEdit;
        export let removeTeacher;
        export let removeSubject;
        export let removeClass;
        export let removeRoom;
        export let downloadRefsJSON;
        export let importRefsJSON;
</script>

<div class="cards">
        <section class="card">
                <div class="card-head">
                        <h2>Учителя</h2>
                        <div class="csvbar">
                                <button class="mini" on:click={() => downloadRefsJSON('teachers')}>⬇ JSON</button>
                                <label class="mini file">⬆<input type="file" accept=".json" on:change={(e) => importRefsJSON('teachers', e)} /></label>
                        </div>
                </div>
                <div class="row">
                        <input bind:value={t.name} placeholder="Имя" />
                        <input bind:value={t.short_name} placeholder="Кратко" />
                        <button class="primary" on:click={addTeacher}>+</button>
                </div>
                <ul class="list">{#each teachers as x}<li>
                        {#if editing && editing.kind === "teacher" && editing.id === x.id}
                                <input class="edit" bind:value={x.name} placeholder="Имя" />
                                <input class="edit w-s" bind:value={x.short_name} placeholder="Кратко" />
                                <button class="primary sm" on:click={() => saveEdit("teacher", x)} title="Сохранить">✓</button>
                                <button class="sm" on:click={cancelEdit} title="Отмена">✗</button>
                        {:else}
                                <span class="li-text">{x.name} <small>({x.short_name})</small></span>
                                <button class="ghost-ico sm" on:click={() => startEdit("teacher", x.id)} title="Редактировать">✎</button>
                                <button class="danger sm" on:click={() => removeTeacher(x.id)}>✕</button>
                        {/if}
                </li>{/each}</ul>
        </section>

        <section class="card">
                <div class="card-head">
                        <h2>Предметы</h2>
                        <div class="csvbar">
                                <button class="mini" on:click={() => downloadRefsJSON('subjects')}>⬇ JSON</button>
                                <label class="mini file">⬆<input type="file" accept=".json" on:change={(e) => importRefsJSON('subjects', e)} /></label>
                        </div>
                </div>
                <div class="row">
                        <input bind:value={s.name} placeholder="Название" />
                        <input bind:value={s.short_name} placeholder="Кратко" />
                        <button class="primary" on:click={addSubject}>+</button>
                </div>
                <ul class="list">{#each subjects as x}<li>
                        {#if editing && editing.kind === "subject" && editing.id === x.id}
                                <input class="edit" bind:value={x.name} placeholder="Название" />
                                <input class="edit w-s" bind:value={x.short_name} placeholder="Кратко" />
                                <button class="primary sm" on:click={() => saveEdit("subject", x)} title="Сохранить">✓</button>
                                <button class="sm" on:click={cancelEdit} title="Отмена">✗</button>
                        {:else}
                                <span class="li-text">{x.name} <small>({x.short_name})</small></span>
                                <button class="ghost-ico sm" on:click={() => startEdit("subject", x.id)} title="Редактировать">✎</button>
                                <button class="danger sm" on:click={() => removeSubject(x.id)}>✕</button>
                        {/if}
                </li>{/each}</ul>
        </section>

        <section class="card">
                <div class="card-head">
                        <h2>Классы</h2>
                        <div class="csvbar">
                                <button class="mini" on:click={() => downloadRefsJSON('classes')}>⬇ JSON</button>
                                <label class="mini file">⬆<input type="file" accept=".json" on:change={(e) => importRefsJSON('classes', e)} /></label>
                        </div>
                </div>
                <div class="row">
                        <input bind:value={c.name} placeholder="10А" />
                        <select bind:value={c.room_id} title="Домашний кабинет класса"><option value={0}>— без кабинета —</option>{#each rooms as rm}<option value={rm.id}>{rm.name}</option>{/each}</select>
                        <select bind:value={c.subgroup_of}><option value={null}>— целый класс —</option>{#each classes as x}<option value={x.id}>{x.name} (подгруппа)</option>{/each}</select>
                        <button class="primary" on:click={addClass}>+</button>
                </div>
                <ul class="list">{#each classes as x}<li>
                        {#if editing && editing.kind === "class" && editing.id === x.id}
                                <input class="edit" bind:value={x.name} placeholder="10А" />
                                <select class="edit" bind:value={x.room_id} title="Домашний кабинет"><option value={0}>— без кабинета —</option>{#each rooms as rm}<option value={rm.id}>{rm.name}</option>{/each}</select>
                                <button class="primary sm" on:click={() => saveEdit("class", x)} title="Сохранить">✓</button>
                                <button class="sm" on:click={cancelEdit} title="Отмена">✗</button>
                        {:else}
                                <span class="li-text">{x.name}{#if x.room_id}<span class="muted"> — каб. {rooms.find((rm) => rm.id === x.room_id)?.name || ""}</span>{/if}{x.subgroup_of ? " · подгруппа" : ""}</span>
                                <button class="ghost-ico sm" on:click={() => startEdit("class", x.id)} title="Редактировать">✎</button>
                                <button class="danger sm" on:click={() => removeClass(x.id)}>✕</button>
                        {/if}
                </li>{/each}</ul>
        </section>

        <section class="card">
                <div class="card-head">
                        <h2>Кабинеты</h2>
                        <div class="csvbar">
                                <button class="mini" on:click={() => downloadRefsJSON('rooms')}>⬇ JSON</button>
                                <label class="mini file">⬆<input type="file" accept=".json" on:change={(e) => importRefsJSON('rooms', e)} /></label>
                        </div>
                </div>
                <div class="row">
                        <input bind:value={r.name} placeholder="301" />
                        <button class="primary" on:click={addRoom}>+</button>
                </div>
                <ul class="list">{#each rooms as x}<li>
                        {#if editing && editing.kind === "room" && editing.id === x.id}
                                <input class="edit" bind:value={x.name} placeholder="301" />
                                <button class="primary sm" on:click={() => saveEdit("room", x)} title="Сохранить">✓</button>
                                <button class="sm" on:click={cancelEdit} title="Отмена">✗</button>
                        {:else}
                                <span class="li-text">{x.name}</span>
                                <button class="ghost-ico sm" on:click={() => startEdit("room", x.id)} title="Редактировать">✎</button>
                                <button class="danger sm" on:click={() => removeRoom(x.id)}>✕</button>
                        {/if}
                </li>{/each}</ul>
        </section>
</div>
