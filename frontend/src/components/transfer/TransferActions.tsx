import { Check, FolderOpen, Plus, Upload, X } from 'lucide-react';
import { OpenFolder } from '../../wailsjs/go/ui/App';
import { Button } from '../ui/Button';
import type { TransferController } from '../../hooks/useTransferController';
import type { PeerRole } from '../../types/navigation';

interface TransferActionsProps {
  role: PeerRole;
  controller: TransferController;
}

/** The actions that make sense for the current phase — nothing more. */
export function TransferActions({ role, controller }: TransferActionsProps) {
  const { phase, completed, isPicking, isResponding, chooseFile, respond, startOver } = controller;
  const sending = role === 'send';

  if (sending && (phase === 'idle' || phase === 'declined' || phase === 'failed')) {
    return (
      <Button variant="primary" size="lg" icon={phase === 'idle' ? Upload : Plus} loading={isPicking} onClick={chooseFile} autoFocus>
        {phase === 'idle' ? 'Choose a file' : 'Choose another file'}
      </Button>
    );
  }

  if (sending && phase === 'completed') {
    return (
      <Button variant="primary" size="lg" icon={Plus} loading={isPicking} onClick={chooseFile} autoFocus>
        Send another file
      </Button>
    );
  }

  if (!sending && phase === 'deciding') {
    return (
      <div className="flex gap-3">
        <Button variant="danger" size="lg" icon={X} disabled={isResponding} onClick={() => respond(false)}>
          Decline
        </Button>
        <Button variant="primary" size="lg" icon={Check} loading={isResponding} onClick={() => respond(true)} autoFocus>
          Accept file
        </Button>
      </div>
    );
  }

  if (!sending && phase === 'completed' && completed) {
    return (
      <div className="flex gap-3">
        <Button variant="secondary" size="lg" onClick={startOver}>
          Done
        </Button>
        <Button variant="primary" size="lg" icon={FolderOpen} onClick={() => OpenFolder(completed.destPath).catch(() => {})} autoFocus>
          Show in folder
        </Button>
      </div>
    );
  }

  if (!sending && (phase === 'declined' || phase === 'failed')) {
    return (
      <Button variant="secondary" size="lg" onClick={startOver} autoFocus>
        Wait for another file
      </Button>
    );
  }

  return null;
}
